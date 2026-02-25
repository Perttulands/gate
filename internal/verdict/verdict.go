package verdict

import "time"

// GateResult is the outcome of a single gate check.
type GateResult struct {
	Name       string `json:"name"`
	Pass       bool   `json:"pass"`
	Skipped    bool   `json:"skipped,omitempty"`
	Output     string `json:"output,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	Findings   *Findings `json:"findings,omitempty"`
}

// Findings holds counts of issues by severity.
type Findings struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

// Verdict is the final output of a gate check run.
type Verdict struct {
	Pass     bool         `json:"pass"`
	Score    float64      `json:"score"`
	Level    string       `json:"level"`
	Citizen  string       `json:"citizen"`
	Repo     string       `json:"repo"`
	Gates    []GateResult `json:"gates"`
	ExitCode int          `json:"exit_code"`
	Bead     string       `json:"bead,omitempty"`
}

// ComputeScore calculates a quality score from gate results.
// The score is the ratio of passing gates to applicable (non-skipped) gates.
// If all gates are skipped, the score is 1.0 (nothing to fail on).
func ComputeScore(gates []GateResult) float64 {
	var applicable, passed int
	for _, g := range gates {
		if g.Skipped {
			continue
		}
		applicable++
		if g.Pass {
			passed++
		}
	}
	if applicable == 0 {
		return 1.0
	}
	return float64(passed) / float64(applicable)
}

// ExitPass means all gates passed.
const ExitPass = 0

// ExitFail means one or more gates failed.
const ExitFail = 1

// ExitReview means warnings present but no hard failures.
const ExitReview = 2

// TimedRun executes fn and returns the result with duration filled in.
func TimedRun(name string, fn func() (bool, string, error)) GateResult {
	start := time.Now()
	pass, output, err := fn()
	dur := time.Since(start).Milliseconds()
	if err != nil {
		return GateResult{Name: name, Pass: false, Output: err.Error(), DurationMs: dur}
	}
	return GateResult{Name: name, Pass: pass, Output: output, DurationMs: dur}
}
