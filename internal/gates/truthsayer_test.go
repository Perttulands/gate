package gates

import (
	"testing"

	"polis/gate/internal/verdict"
)

func TestParseTruthsayerOutput_SummaryLine(t *testing.T) {
	output := `ERROR   silent-fallback.ignored-error
  file.go:10
  > bad code

WARN    bad-defaults.unvalidated-env-go
  file.go:20
  > os.Getenv

INFO    trace-gaps.no-stderr-capture
  file.go:30
  > exec.Command

──────────────────────────────────────────────────
Summary: 1 error, 1 warning, 1 info (5 files scanned in 2ms)
Categories: bad-defaults: 1`

	f := parseTruthsayerOutput(output)
	want := verdict.Findings{Errors: 1, Warnings: 1, Info: 1}
	if f != want {
		t.Errorf("got %+v, want %+v", f, want)
	}
}

func TestParseTruthsayerOutput_MultipleErrors(t *testing.T) {
	output := `Summary: 3 errors, 2 warnings, 10 info (11 files scanned in 3ms)
Categories: bad-defaults: 10`

	f := parseTruthsayerOutput(output)
	if f.Errors != 3 || f.Warnings != 2 || f.Info != 10 {
		t.Errorf("got %+v, want errors=3 warnings=2 info=10", f)
	}
}

func TestParseTruthsayerOutput_ZeroErrors(t *testing.T) {
	output := `Summary: 0 errors, 0 warnings, 0 info (5 files scanned in 1ms)`
	f := parseTruthsayerOutput(output)
	if f.Errors != 0 || f.Warnings != 0 || f.Info != 0 {
		t.Errorf("got %+v, want all zeros", f)
	}
}

func TestParseTruthsayerOutput_FallbackCounting(t *testing.T) {
	// No summary line — should fall back to counting prefixes
	output := `ERROR   something.bad
  file.go:1
ERROR   another.bad
  file.go:2
WARN    minor.issue
  file.go:3`

	f := parseTruthsayerOutput(output)
	if f.Errors != 2 {
		t.Errorf("expected 2 errors from fallback, got %d", f.Errors)
	}
	if f.Warnings != 1 {
		t.Errorf("expected 1 warning from fallback, got %d", f.Warnings)
	}
}

func TestParseTruthsayerOutput_EmptyOutput(t *testing.T) {
	f := parseTruthsayerOutput("")
	if f.Errors != 0 || f.Warnings != 0 || f.Info != 0 {
		t.Errorf("expected all zeros for empty output, got %+v", f)
	}
}
