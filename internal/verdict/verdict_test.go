package verdict

import (
	"testing"
	"time"
)

func TestTimedRun_Pass(t *testing.T) {
	r := TimedRun("test-gate", func() (bool, string, error) {
		return true, "all good", nil
	})
	if !r.Pass {
		t.Fatal("expected pass")
	}
	if r.Name != "test-gate" {
		t.Fatalf("expected name test-gate, got %s", r.Name)
	}
	if r.Output != "all good" {
		t.Fatalf("expected output 'all good', got %q", r.Output)
	}
}

func TestTimedRun_Fail(t *testing.T) {
	r := TimedRun("test-gate", func() (bool, string, error) {
		return false, "broken", nil
	})
	if r.Pass {
		t.Fatal("expected fail")
	}
}

func TestTimedRun_Error(t *testing.T) {
	r := TimedRun("test-gate", func() (bool, string, error) {
		return false, "", &time.ParseError{Message: "bang"}
	})
	if r.Pass {
		t.Fatal("expected fail on error")
	}
	if r.Output == "" {
		t.Fatal("expected error in output")
	}
}
