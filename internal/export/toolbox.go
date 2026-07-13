package export

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

const (
	DefaultToolboxImage = "registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16"
)

// containerRuntimes lists runtimes tried when none is configured (docker first for broader client support).
var containerRuntimes = []string{"docker", "podman"}

var ErrToolboxFailed = errors.New("3scale toolbox product export failed")

type ProductExporter interface {
	ExportProduct(ctx context.Context, adminURL, token, systemName string) ([]byte, error)
}

// CommandRunner executes external processes. Inject a mock in tests to avoid real exec.
type CommandRunner interface {
	Run(ctx context.Context, command string, args []string) (stdout, stderr []byte, err error)
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, command string, args []string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

type ToolboxOptions struct {
	Runtime string // podman or docker; empty auto-detects
	Image   string
	// NativeBinary, if set, runs the local 3scale binary instead of a container.
	NativeBinary string
	// CertFile mounts a CA/cert for toolbox TLS (SSL_CERT_FILE in container).
	CertFile string
	// Insecure passes -k to toolbox to skip TLS verification (lab tenants).
	Insecure bool
	// CommandRunner overrides process execution (defaults to os/exec).
	CommandRunner CommandRunner
}

type Toolbox struct {
	runtime      string
	image        string
	nativeBinary string
	certFile     string
	insecure     bool
	runner       CommandRunner
}

func NewToolbox(opts ToolboxOptions) (*Toolbox, error) {
	t := &Toolbox{
		runtime:      strings.TrimSpace(opts.Runtime),
		image:        strings.TrimSpace(opts.Image),
		nativeBinary: strings.TrimSpace(opts.NativeBinary),
		certFile:     strings.TrimSpace(opts.CertFile),
		insecure:     opts.Insecure,
		runner:       opts.CommandRunner,
	}
	if t.runner == nil {
		t.runner = execCommandRunner{}
	}
	if t.image == "" {
		t.image = DefaultToolboxImage
	}
	if t.nativeBinary == "" {
		runtime, err := resolveContainerRuntime(t.runtime)
		if err != nil {
			return nil, err
		}
		t.runtime = runtime
	}
	return t, nil
}

func resolveContainerRuntime(preferred string) (string, error) {
	if preferred != "" {
		if _, err := exec.LookPath(preferred); err != nil {
			return "", fmt.Errorf("toolbox runtime %q not found in PATH", preferred)
		}
		return preferred, nil
	}
	for _, candidate := range containerRuntimes {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("docker or podman is required for the 3scale toolbox container image (see Red Hat docs)")
}

func (t *Toolbox) ExportProduct(ctx context.Context, adminURL, token, systemName string) ([]byte, error) {
	systemName = strings.TrimSpace(systemName)
	if systemName == "" {
		return nil, errors.New("system name is required for product export")
	}
	remoteURL, err := buildRemoteURL(adminURL, token)
	if err != nil {
		return nil, err
	}

	if t.nativeBinary != "" {
		return t.runNative(ctx, remoteURL, systemName)
	}
	return t.runContainer(ctx, remoteURL, systemName)
}

func buildRemoteURL(adminURL, token string) (string, error) {
	adminURL = strings.TrimRight(strings.TrimSpace(adminURL), "/")
	token = strings.TrimSpace(token)
	if adminURL == "" || token == "" {
		return "", errors.New("admin URL and token are required for toolbox export")
	}
	u, err := url.Parse(adminURL)
	if err != nil {
		return "", err
	}
	u.User = url.User(token)
	return u.String(), nil
}

func (t *Toolbox) runNative(ctx context.Context, remoteURL, systemName string) ([]byte, error) {
	return t.runCommand(ctx, t.nativeBinary, t.toolboxProductArgs(remoteURL, systemName))
}

func (t *Toolbox) runContainer(ctx context.Context, remoteURL, systemName string) ([]byte, error) {
	args := []string{"run", "--rm"}
	if t.certFile != "" {
		args = append(args,
			"--env", "SSL_CERT_FILE=/tmp/3scale-toolbox-cert.pem",
			"-v", t.certFile+":/tmp/3scale-toolbox-cert.pem:ro",
		)
	}
	args = append(args, t.image, "3scale")
	args = append(args, t.toolboxProductArgs(remoteURL, systemName)...)
	return t.runCommand(ctx, t.runtime, args)
}

func (t *Toolbox) toolboxProductArgs(remoteURL, systemName string) []string {
	args := make([]string, 0, 5)
	if t.insecure {
		args = append(args, "-k")
	}
	args = append(args, "product", "export", remoteURL, systemName)
	return args
}

func (t *Toolbox) runCommand(ctx context.Context, command string, args []string) ([]byte, error) {
	stdout, stderr, err := t.runner.Run(ctx, command, args)
	if err != nil {
		msg := strings.TrimSpace(string(stderr))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%w: %s", ErrToolboxFailed, msg)
	}

	out := bytes.TrimSpace(stdout)
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: empty output from %s", ErrToolboxFailed, command)
	}
	return out, nil
}
