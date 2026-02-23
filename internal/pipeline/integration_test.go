package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"polis/gate/internal/verdict"
)

func TestIntegration_PassingGoProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module passing\n\ngo 1.21\n")
	writeFile(t, dir, "main.go", "package main\nfunc main() {}\n")
	writeFile(t, dir, "main_test.go", `package main
import "testing"
func TestNothing(t *testing.T) {}
`)

	v := Run(context.Background(), dir, LevelQuick, "integration-tester")
	if !v.Pass {
		t.Fatalf("expected pass for valid Go project, gates: %+v", v.Gates)
	}
	if v.ExitCode != verdict.ExitPass {
		t.Errorf("expected exit code 0, got %d", v.ExitCode)
	}
}

func TestIntegration_FailingGoProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module failing\n\ngo 1.21\n")
	writeFile(t, dir, "main.go", "package main\nfunc main() {}\n")
	writeFile(t, dir, "main_test.go", `package main
import "testing"
func TestFail(t *testing.T) { t.Fatal("intentional failure") }
`)

	v := Run(context.Background(), dir, LevelQuick, "integration-tester")
	if v.Pass {
		t.Fatal("expected fail for project with failing test")
	}
	if v.ExitCode != verdict.ExitFail {
		t.Errorf("expected exit code 1, got %d", v.ExitCode)
	}

	// Verify the tests gate specifically failed
	for _, g := range v.Gates {
		if g.Name == "tests" {
			if g.Pass {
				t.Error("tests gate should have failed")
			}
			return
		}
	}
	t.Error("tests gate not found in results")
}

func TestIntegration_VerdictJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module jsontest\n\ngo 1.21\n")
	writeFile(t, dir, "main.go", "package main\nfunc main() {}\n")

	v := Run(context.Background(), dir, LevelQuick, "json-tester")

	// Serialize and deserialize — verify JSON roundtrip
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal verdict: %v", err)
	}

	var parsed verdict.Verdict
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal verdict: %v", err)
	}

	if parsed.Pass != v.Pass {
		t.Errorf("pass mismatch: got %v, want %v", parsed.Pass, v.Pass)
	}
	if parsed.Level != v.Level {
		t.Errorf("level mismatch: got %q, want %q", parsed.Level, v.Level)
	}
	if parsed.Citizen != v.Citizen {
		t.Errorf("citizen mismatch: got %q, want %q", parsed.Citizen, v.Citizen)
	}
	if len(parsed.Gates) != len(v.Gates) {
		t.Errorf("gates count mismatch: got %d, want %d", len(parsed.Gates), len(v.Gates))
	}
}

func TestIntegration_GracefulSkip_StandardLevel(t *testing.T) {
	// Standard level includes truthsayer and ubs — they should gracefully skip
	// if not available (or pass if available).
	dir := t.TempDir()

	v := Run(context.Background(), dir, LevelStandard, "skip-tester")

	hasTruthsayer, hasUBS := false, false
	for _, g := range v.Gates {
		if g.Name == "truthsayer" {
			hasTruthsayer = true
			// Should either pass (if installed) or be skipped
			if !g.Pass {
				t.Errorf("truthsayer should pass or skip, got fail: %s", g.Output)
			}
		}
		if g.Name == "ubs" {
			hasUBS = true
			if !g.Pass {
				t.Errorf("ubs should pass or skip, got fail: %s", g.Output)
			}
		}
	}
	if !hasTruthsayer {
		t.Error("standard level should include truthsayer gate")
	}
	if !hasUBS {
		t.Error("standard level should include ubs gate")
	}
}

func TestIntegration_GoVetFailure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module vetfail\n\ngo 1.21\n")
	// Code that go vet will flag: Printf with wrong format
	writeFile(t, dir, "main.go", `package main
import "fmt"
func main() {
	var x int = 42
	fmt.Printf("%s\n", x)
}
`)

	v := Run(context.Background(), dir, LevelQuick, "vet-tester")

	lintFailed := false
	for _, g := range v.Gates {
		if g.Name == "lint:go vet" && !g.Pass {
			lintFailed = true
		}
	}
	if !lintFailed {
		t.Error("expected lint:go vet to fail on Printf format mismatch")
	}
}

func TestIntegration_FindingsInResult(t *testing.T) {
	dir := t.TempDir()

	v := Run(context.Background(), dir, LevelStandard, "findings-tester")

	for _, g := range v.Gates {
		if g.Name == "truthsayer" && g.Findings != nil {
			// Findings struct should have been populated
			if g.Findings.Errors < 0 {
				t.Error("findings errors should be >= 0")
			}
		}
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}
