package gates

import (
	"testing"
)

func TestParseUBSOutput_WithErrors(t *testing.T) {
	output := `UBS Meta-Runner v5.0.7
Project: /home/user/project
Detected: golang
✗ failed to verify module golang: checksum mismatch
✗ failed to ensure module for golang`

	f := parseUBSOutput(output)
	if f.Errors != 2 {
		t.Errorf("expected 2 errors, got %d", f.Errors)
	}
}

func TestParseUBSOutput_Clean(t *testing.T) {
	output := `UBS Meta-Runner v5.0.7
Project: /home/user/project
Detected: golang
✓ all checks passed`

	f := parseUBSOutput(output)
	if f.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", f.Errors)
	}
}

func TestParseUBSOutput_WithWarnings(t *testing.T) {
	output := `⚠ possible issue detected
⚠ another warning
✗ critical failure`

	f := parseUBSOutput(output)
	if f.Warnings != 2 {
		t.Errorf("expected 2 warnings, got %d", f.Warnings)
	}
	if f.Errors != 1 {
		t.Errorf("expected 1 error, got %d", f.Errors)
	}
}

func TestParseUBSOutput_Empty(t *testing.T) {
	f := parseUBSOutput("")
	if f.Errors != 0 || f.Warnings != 0 {
		t.Errorf("expected all zeros for empty output, got %+v", f)
	}
}
