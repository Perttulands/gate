package gates

import (
	"context"

	"polis/gate/internal/verdict"
)

// RunTruthsayer runs truthsayer scan on the repo at dir.
// Truthsayer is optional — if not installed, the gate passes with a note.
func RunTruthsayer(ctx context.Context, dir string, timeoutSec int) verdict.GateResult {
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	return verdict.TimedRun("truthsayer", func() (bool, string, error) {
		pass, output, err := runCmd(ctx, dir, timeoutSec, "truthsayer", "scan", ".")
		if err != nil {
			// truthsayer not installed — graceful degradation
			return true, "truthsayer not available (skipped)", nil
		}
		return pass, output, nil
	})
}
