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
	Level    string       `json:"level"`
	Citizen  string       `json:"citizen"`
	Repo     string       `json:"repo"`
	Gates    []GateResult `json:"gates"`
	ExitCode int          `json:"exit_code"`
	Bead     string       `json:"bead,omitempty"`
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
