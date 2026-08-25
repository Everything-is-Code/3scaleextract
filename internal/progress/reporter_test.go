package progress

import (
	"bytes"
	"strings"
	"testing"
)

func TestNopReporter(t *testing.T) {
	var buf bytes.Buffer
	rep := New(&buf, true, true)
	rep.Phase("phase")
	rep.Item(1, 2, "item")
	rep.Warning("warn")
	rep.Verbose("verbose")
	rep.Done("done")
	if buf.Len() != 0 {
		t.Fatalf("quiet reporter wrote %q", buf.String())
	}
}

func TestStderrReporterNormal(t *testing.T) {
	var buf bytes.Buffer
	rep := New(&buf, false, false)
	rep.Phase("Listing products")
	rep.Item(2, 5, "payments")
	rep.Warning("skipped proxy")
	rep.Verbose("hidden")
	rep.Done("complete")

	out := buf.String()
	if !strings.Contains(out, "→ Listing products") {
		t.Fatalf("missing phase: %s", out)
	}
	if !strings.Contains(out, "→ [2/5] payments") {
		t.Fatalf("missing item: %s", out)
	}
	if !strings.Contains(out, "⚠ skipped proxy") {
		t.Fatalf("missing warning: %s", out)
	}
	if strings.Contains(out, "hidden") {
		t.Fatalf("verbose should be suppressed: %s", out)
	}
	if !strings.Contains(out, "✓ complete") {
		t.Fatalf("missing done: %s", out)
	}
}

func TestStderrReporterVerbose(t *testing.T) {
	var buf bytes.Buffer
	rep := New(&buf, false, true)
	rep.Verbose("toolbox detail")
	if !strings.Contains(buf.String(), "toolbox detail") {
		t.Fatalf("expected verbose line: %s", buf.String())
	}
}
