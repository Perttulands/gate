package pipeline

import (
	"context"
	"path/filepath"

	"polis/gate/internal/gates"
	"polis/gate/internal/verdict"
)

// Level controls how thorough the gate check is.
const (
	LevelQuick    = "quick"
	LevelStandard = "standard"
	LevelDeep     = "deep"
)

// ValidLevel returns true if level is a known level string.
func ValidLevel(level string) bool {
	switch level {
	case LevelQuick, LevelStandard, LevelDeep:
		return true
	}
	return false
}

// Run executes the gate pipeline at the given level and returns a verdict.
func Run(ctx context.Context, repoPath, level, citizen string) verdict.Verdict {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		setupGates := []verdict.GateResult{{Name: "setup", Pass: false, Output: err.Error()}}
		return verdict.Verdict{
			Pass:     false,
			Score:    verdict.ComputeScore(setupGates),
			Level:    level,
			Citizen:  citizen,
			Repo:     repoPath,
			ExitCode: verdict.ExitFail,
			Gates:    setupGates,
		}
	}

	repoName := filepath.Base(absPath)
	var results []verdict.GateResult

	// Quick: tests + lint
	testResult := gates.RunTests(ctx, absPath, 120)
	results = append(results, testResult)

	lintResults := gates.RunLint(ctx, absPath, 60)
	results = append(results, lintResults...)

	// Standard: + truthsayer + ubs
	if level == LevelStandard || level == LevelDeep {
		if level == LevelStandard {
			// PR-friendly gate: changed-lines/files focus.
			tsResult := gates.RunTruthsayerCI(ctx, absPath, 60)
			results = append(results, tsResult)

			ubsResult := gates.RunUBSDiff(ctx, absPath, 60)
			results = append(results, ubsResult)
		} else {
			// Deep gate: full scans.
			tsResult := gates.RunTruthsayer(ctx, absPath, 60)
			results = append(results, tsResult)

			ubsResult := gates.RunUBS(ctx, absPath, 60)
			results = append(results, ubsResult)
		}
	}

	// Deep: + risk scoring (placeholder for now)
	if level == LevelDeep {
		riskResult := verdict.GateResult{Name: "risk", Pass: true, Output: "risk scoring not yet implemented", DurationMs: 0}
		results = append(results, riskResult)
	}

	// Compute overall pass/fail
	allPass := true
	for _, r := range results {
		if !r.Pass {
			allPass = false
			break
		}
	}

	exitCode := verdict.ExitPass
	if !allPass {
		exitCode = verdict.ExitFail
	}

	return verdict.Verdict{
		Pass:     allPass,
		Score:    verdict.ComputeScore(results),
		Level:    level,
		Citizen:  citizen,
		Repo:     repoName,
		Gates:    results,
		ExitCode: exitCode,
	}
}
