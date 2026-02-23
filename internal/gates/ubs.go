package gates

import (
	"context"
	"fmt"
	"strings"
	"time"

	"polis/gate/internal/verdict"
)

// RunUBS runs ubs build health check on the repo at dir.
// UBS is optional — if not installed, the gate passes with skipped=true.
// Pass criteria: no critical-level failures in output.
func RunUBS(ctx context.Context, dir string, timeoutSec int) verdict.GateResult {
	if timeoutSec <= 0 {
		timeoutSec = 60
	}

	start := time.Now()
	pass, output, err := runCmd(ctx, dir, timeoutSec, "ubs", ".")
	dur := time.Since(start).Milliseconds()

	if err != nil {
		return verdict.GateResult{
			Name:       "ubs",
			Pass:       true,
			Skipped:    true,
			Output:     "ubs not available (skipped)",
			DurationMs: dur,
		}
	}

	findings := parseUBSOutput(output)

	summary := fmt.Sprintf("pass=%v, %d errors, %d warnings", pass, findings.Errors, findings.Warnings)
	if !pass {
		summary = output
	}

	return verdict.GateResult{
		Name:       "ubs",
		Pass:       pass,
		Output:     summary,
		DurationMs: dur,
		Findings:   &findings,
	}
}

// parseUBSOutput counts error and warning markers in UBS output.
func parseUBSOutput(output string) verdict.Findings {
	var f verdict.Findings
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "✗") {
			f.Errors++
		} else if strings.HasPrefix(trimmed, "⚠") {
			f.Warnings++
		}
	}
	return f
}
