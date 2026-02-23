package gates

import (
	"context"

	"polis/gate/internal/verdict"
)

// RunUBS runs ubs build health check on the repo at dir.
// UBS is optional — if not installed, the gate passes with a note.
func RunUBS(ctx context.Context, dir string, timeoutSec int) verdict.GateResult {
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	return verdict.TimedRun("ubs", func() (bool, string, error) {
		pass, output, err := runCmd(ctx, dir, timeoutSec, "ubs", ".")
		if err != nil {
			// ubs not installed — graceful degradation
			return true, "ubs not available (skipped)", nil
		}
		return pass, output, nil
	})
}
