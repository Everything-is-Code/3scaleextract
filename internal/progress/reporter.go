package progress

import (
	"fmt"
	"io"
	"os"
)

// Reporter emits human-readable export progress to stderr (or any Writer).
type Reporter interface {
	Phase(message string)
	Item(current, total int, label string)
	Warning(message string)
	Verbose(message string)
	Done(message string)
}

type nopReporter struct{}

func (nopReporter) Phase(string)    {}
func (nopReporter) Item(int, int, string) {}
func (nopReporter) Warning(string)  {}
func (nopReporter) Verbose(string)  {}
func (nopReporter) Done(string)     {}

// Nop returns a Reporter that discards all messages.
func Nop() Reporter { return nopReporter{} }

type stderrReporter struct {
	w       io.Writer
	verbose bool
}

// New builds a stderr progress reporter. When quiet is true, returns Nop().
func New(w io.Writer, quiet, verbose bool) Reporter {
	if quiet {
		return Nop()
	}
	if w == nil {
		w = os.Stderr
	}
	return &stderrReporter{w: w, verbose: verbose}
}

func (r *stderrReporter) Phase(message string) {
	fmt.Fprintf(r.w, "→ %s\n", message)
}

func (r *stderrReporter) Item(current, total int, label string) {
	if total > 0 {
		fmt.Fprintf(r.w, "→ [%d/%d] %s\n", current, total, label)
	} else {
		fmt.Fprintf(r.w, "→ %s\n", label)
	}
}

func (r *stderrReporter) Warning(message string) {
	fmt.Fprintf(r.w, "⚠ %s\n", message)
}

func (r *stderrReporter) Verbose(message string) {
	if r.verbose {
		fmt.Fprintf(r.w, "  %s\n", message)
	}
}

func (r *stderrReporter) Done(message string) {
	fmt.Fprintf(r.w, "✓ %s\n", message)
}

// OrNop returns r when non-nil, otherwise Nop().
func OrNop(r Reporter) Reporter {
	if r == nil {
		return Nop()
	}
	return r
}
