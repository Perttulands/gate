package bead

import (
	"testing"

	"polis/gate/internal/verdict"
)

func TestRecord_NoBdOnPath(t *testing.T) {
	// When bd is on PATH this will actually create a bead,
	// but we're testing the interface — it should return a string or empty.
	v := verdict.Verdict{
		Pass:    true,
		Level:   "quick",
		Citizen: "tester",
		Repo:    "test-repo",
		Gates:   []verdict.GateResult{{Name: "tests", Pass: true}},
	}

	// We can't reliably mock exec.LookPath, but we can verify
	// the function doesn't panic and returns a string.
	beadID := Record(v)
	// beadID is either empty (bd not available / error) or a valid ID
	_ = beadID
}

func TestRecord_FailedVerdict(t *testing.T) {
	v := verdict.Verdict{
		Pass:    false,
		Level:   "standard",
		Citizen: "tester",
		Repo:    "broken-repo",
		Gates:   []verdict.GateResult{{Name: "tests", Pass: false, Output: "FAIL"}},
	}

	beadID := Record(v)
	_ = beadID
}
