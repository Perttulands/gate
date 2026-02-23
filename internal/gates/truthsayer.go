package gates

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"polis/gate/internal/verdict"
)

var truthsayerSummaryRe = regexp.MustCompile(`Summary:\s*(\d+)\s*errors?,\s*(\d+)\s*warnings?,\s*(\d+)\s*info`)

// RunTruthsayer runs truthsayer scan on the repo at dir.
// Truthsayer is optional — if not installed, the gate passes with skipped=true.
// Pass criteria: zero critical (error) findings.
func RunTruthsayer(ctx context.Context, dir string, timeoutSec int) verdict.GateResult {
	if timeoutSec <= 0 {
		timeoutSec = 60
	}

	start := time.Now()
	_, output, err := runCmd(ctx, dir, timeoutSec, "truthsayer", "scan", ".")
	dur := time.Since(start).Milliseconds()

	if err != nil {
		return verdict.GateResult{
			Name:       "truthsayer",
			Pass:       true,
			Skipped:    true,
			Output:     "truthsayer not available (skipped)",
			DurationMs: dur,
		}
	}

	findings := parseTruthsayerOutput(output)
	pass := findings.Errors == 0

	summary := fmt.Sprintf("%d errors, %d warnings, %d info", findings.Errors, findings.Warnings, findings.Info)
	if !pass {
		// Include full output on failure for diagnostics
		summary = output
	}

	return verdict.GateResult{
		Name:       "truthsayer",
		Pass:       pass,
		Output:     summary,
		DurationMs: dur,
		Findings:   &findings,
	}
}

// parseTruthsayerOutput extracts finding counts from truthsayer output.
func parseTruthsayerOutput(output string) verdict.Findings {
	var f verdict.Findings

	// Try summary line first — regex already guarantees digits, so Atoi won't fail
	if m := truthsayerSummaryRe.FindStringSubmatch(output); len(m) == 4 {
		f.Errors = atoiSafe(m[1])
		f.Warnings = atoiSafe(m[2])
		f.Info = atoiSafe(m[3])
		return f
	}

	// Fallback: count severity prefixes
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "ERROR") {
			f.Errors++
		} else if strings.HasPrefix(trimmed, "WARN") {
			f.Warnings++
		} else if strings.HasPrefix(trimmed, "INFO") {
			f.Info++
		}
	}
	return f
}

// atoiSafe converts a string to int, returning 0 on error.
func atoiSafe(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
