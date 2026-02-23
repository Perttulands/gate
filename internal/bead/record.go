package bead

import (
	"fmt"
	"os/exec"
	"strings"

	"polis/gate/internal/verdict"
)

// Record creates a bead for the given verdict using bd.
// Returns the bead ID, or empty string if bd is not available.
func Record(v verdict.Verdict) string {
	if _, err := exec.LookPath("bd"); err != nil {
		return ""
	}

	status := "pass"
	if !v.Pass {
		status = "fail"
	}

	title := fmt.Sprintf("%s gate %s: %s", v.Repo, v.Level, status)
	labels := fmt.Sprintf("tool:gate,status:%s,repo:%s,level:%s", status, v.Repo, v.Level)

	args := []string{
		"create",
		"--type", "gate",
		"--title", title,
		"--labels", labels,
		"--silent",
	}
	if v.Citizen != "" && v.Citizen != "unknown" {
		args = append(args, "-a", v.Citizen)
	}

	cmd := exec.Command("bd", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
