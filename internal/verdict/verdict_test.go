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

func TestComputeScore_AllPass(t *testing.T) {
	gates := []GateResult{
		{Name: "tests", Pass: true},
		{Name: "lint", Pass: true},
	}
	score := ComputeScore(gates)
	if score != 1.0 {
		t.Errorf("expected 1.0, got %f", score)
	}
}

func TestComputeScore_SomeFail(t *testing.T) {
	gates := []GateResult{
		{Name: "tests", Pass: true},
		{Name: "lint", Pass: false},
	}
	score := ComputeScore(gates)
	if score != 0.5 {
		t.Errorf("expected 0.5, got %f", score)
	}
}

func TestComputeScore_AllFail(t *testing.T) {
	gates := []GateResult{
		{Name: "tests", Pass: false},
		{Name: "lint", Pass: false},
	}
	score := ComputeScore(gates)
	if score != 0.0 {
		t.Errorf("expected 0.0, got %f", score)
	}
}

func TestComputeScore_AllSkipped(t *testing.T) {
	gates := []GateResult{
		{Name: "truthsayer", Pass: true, Skipped: true},
		{Name: "ubs", Pass: true, Skipped: true},
	}
	score := ComputeScore(gates)
	if score != 1.0 {
		t.Errorf("expected 1.0 when all skipped, got %f", score)
	}
}

func TestComputeScore_MixedWithSkipped(t *testing.T) {
	gates := []GateResult{
		{Name: "tests", Pass: true},
		{Name: "lint", Pass: true},
		{Name: "truthsayer", Pass: true, Skipped: true},
	}
	score := ComputeScore(gates)
	if score != 1.0 {
		t.Errorf("expected 1.0 (skipped excluded), got %f", score)
	}
}

func TestComputeScore_Empty(t *testing.T) {
	score := ComputeScore(nil)
	if score != 1.0 {
		t.Errorf("expected 1.0 for empty gates, got %f", score)
	}
}

func TestComputeScore_OneOfThreeFails(t *testing.T) {
	gates := []GateResult{
		{Name: "tests", Pass: true},
		{Name: "lint:go vet", Pass: false},
		{Name: "lint:shellcheck", Pass: true},
	}
	score := ComputeScore(gates)
	want := 2.0 / 3.0
	if score < want-0.001 || score > want+0.001 {
		t.Errorf("expected ~%.4f, got %f", want, score)
	}
}
