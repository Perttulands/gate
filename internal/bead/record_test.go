package bead

import (
	"errors"
	"strings"
	"testing"

	"polis/gate/internal/city"
	"polis/gate/internal/verdict"
)

func TestRecordCity_UsesBRWhenAvailable(t *testing.T) {
	defer resetHooksForTest()

	var gotName string
	var gotArgs []string
	lookPath = func(name string) (string, error) {
		if name == "br" {
			return "/usr/bin/br", nil
		}
		return "", errors.New("missing")
	}
	runCmd = func(name string, args ...string) ([]byte, error) {
		gotName = name
		gotArgs = append([]string{}, args...)
		return []byte("pol-123\n"), nil
	}

	id := RecordCity(city.Verdict{
		Repo:     "relay",
		Status:   "warn",
		ExitCode: city.ExitWarn,
		Summary:  city.Summary{Pass: 2, Skip: 2},
		Checks: []city.CheckResult{
			{Name: "standalone", Status: city.StatusSkip, Detail: "skipped by --skip-standalone"},
		},
	}, "tester")

	if id != "pol-123" {
		t.Fatalf("expected bead id pol-123, got %q", id)
	}
	if gotName != "br" {
		t.Fatalf("expected br command, got %q", gotName)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "gate city: relay (warn)") {
		t.Fatalf("expected city title in args: %v", gotArgs)
	}
	if !strings.Contains(joined, "kind:city") {
		t.Fatalf("expected city labels in args: %v", gotArgs)
	}
}

func TestRecord_UsesBR(t *testing.T) {
	defer resetHooksForTest()

	lookPath = func(name string) (string, error) {
		if name == "br" {
			return "/usr/bin/br", nil
		}
		return "", errors.New("missing")
	}

	var gotName string
	var gotArgs []string
	runCmd = func(name string, args ...string) ([]byte, error) {
		gotName = name
		gotArgs = append([]string{}, args...)
		return []byte("br-9\n"), nil
	}

	id := Record(verdict.Verdict{
		Pass:    false,
		Level:   "standard",
		Citizen: "tester",
		Repo:    "gate",
		Gates:   []verdict.GateResult{{Name: "tests", Pass: false}},
	})

	if id != "br-9" {
		t.Fatalf("expected br-9 id, got %q", id)
	}
	if gotName != "br" {
		t.Fatalf("expected br command, got %q", gotName)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "gate gate") {
		t.Fatalf("expected title in args: %v", gotArgs)
	}
}

func TestRecord_NoBRReturnsEmpty(t *testing.T) {
	defer resetHooksForTest()

	lookPath = func(name string) (string, error) { return "", errors.New("missing") }
	runCmd = func(name string, args ...string) ([]byte, error) {
		t.Fatalf("runCmd should not be called when br unavailable")
		return nil, nil
	}

	id := Record(verdict.Verdict{
		Pass:  true,
		Level: "quick",
		Repo:  "gate",
	})
	if id != "" {
		t.Fatalf("expected empty id, got %q", id)
	}
}

func TestRecord_NoToolReturnsEmpty(t *testing.T) {
	defer resetHooksForTest()

	lookPath = func(name string) (string, error) { return "", errors.New("missing") }
	runCmd = func(name string, args ...string) ([]byte, error) {
		t.Fatalf("runCmd should not be called when tools unavailable")
		return nil, nil
	}

	id := RecordCity(city.Verdict{Repo: "x", Status: "fail"}, "tester")
	if id != "" {
		t.Fatalf("expected empty id, got %q", id)
	}
}

func TestFormatCityDescription_IncludesChecks(t *testing.T) {
	out := formatCityDescription(city.Verdict{
		Repo:     "relay",
		Status:   "fail",
		ExitCode: city.ExitFail,
		Summary:  city.Summary{Pass: 1, Fail: 2, Skip: 1},
		Checks: []city.CheckResult{
			{Name: "boundary", Status: city.StatusPass, Detail: "ok", DurationMs: 3},
			{Name: "split", Status: city.StatusFail, Detail: "missing", DurationMs: 2},
		},
	})
	if !strings.Contains(out, "gate city verdict: fail") {
		t.Fatalf("missing status in description: %q", out)
	}
	if !strings.Contains(out, "- split: fail") {
		t.Fatalf("missing check line in description: %q", out)
	}
}
