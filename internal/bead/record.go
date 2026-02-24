package bead

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"polis/gate/internal/city"
	"polis/gate/internal/verdict"
)

var (
	lookPath = exec.LookPath
	runCmd   = func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).Output()
	}
)

// Record creates a bead for a gate check verdict.
func Record(v verdict.Verdict) string {
	status := "pass"
	if !v.Pass {
		status = "fail"
	}

	title := fmt.Sprintf("%s gate %s: %s", v.Repo, v.Level, status)
	labels := fmt.Sprintf("tool:gate,status:%s,repo:%s,level:%s", status, v.Repo, v.Level)
	description := formatCheckDescription(v)

	if beadID := createWithBR(title, labels, description, v.Citizen); beadID != "" {
		return beadID
	}
	return createWithBD("gate", title, labels, description, v.Citizen)
}

// RecordCity creates a bead for a gate city verdict.
func RecordCity(v city.Verdict, citizen string) string {
	title := fmt.Sprintf("gate city: %s (%s)", v.Repo, v.Status)
	labels := fmt.Sprintf("tool:gate,kind:city,status:%s,repo:%s", v.Status, v.Repo)
	description := formatCityDescription(v)
	if beadID := createWithBR(title, labels, description, citizen); beadID != "" {
		return beadID
	}
	return createWithBD("gate", title, labels, description, citizen)
}

func createWithBR(title, labels, description, citizen string) string {
	if _, err := lookPath("br"); err != nil {
		return ""
	}
	args := []string{
		"create",
		title,
		"-t", "chore",
		"-l", labels,
		"-d", description,
		"--silent",
	}
	if citizen != "" && citizen != "unknown" {
		args = append(args, "-a", citizen)
	}
	out, err := runCmd("br", args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func createWithBD(issueType, title, labels, description, citizen string) string {
	if _, err := lookPath("bd"); err != nil {
		return ""
	}
	args := []string{
		"create",
		"--type", issueType,
		"--title", title,
		"--labels", labels,
		"--description", description,
		"--silent",
	}
	if citizen != "" && citizen != "unknown" {
		args = append(args, "-a", citizen)
	}
	out, err := runCmd("bd", args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func formatCheckDescription(v verdict.Verdict) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("gate check verdict: %s", boolStatus(v.Pass)))
	lines = append(lines, fmt.Sprintf("repo: %s", v.Repo))
	lines = append(lines, fmt.Sprintf("level: %s", v.Level))
	lines = append(lines, "checks:")
	for _, g := range v.Gates {
		status := boolStatus(g.Pass)
		if g.Skipped {
			status = "skip"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s (%dms)", g.Name, status, g.DurationMs))
	}
	return strings.Join(lines, "\n")
}

func formatCityDescription(v city.Verdict) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("gate city verdict: %s", v.Status))
	lines = append(lines, fmt.Sprintf("repo: %s", v.Repo))
	lines = append(lines, fmt.Sprintf("exit_code: %d", v.ExitCode))
	lines = append(lines, fmt.Sprintf("summary: pass=%d fail=%d skip=%d", v.Summary.Pass, v.Summary.Fail, v.Summary.Skip))
	lines = append(lines, "")
	lines = append(lines, "checks:")
	for _, c := range v.Checks {
		lines = append(lines, fmt.Sprintf("- %s: %s (%dms) %s", c.Name, c.Status, c.DurationMs, c.Detail))
	}
	return strings.Join(lines, "\n")
}

func boolStatus(pass bool) string {
	if pass {
		return "pass"
	}
	return "fail"
}

// resetHooksForTest restores package globals changed in tests.
func resetHooksForTest() {
	lookPath = exec.LookPath
	runCmd = func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).Output()
	}
}

// normalizeLabels returns labels sorted lexicographically to simplify assertions.
func normalizeLabels(v string) string {
	parts := strings.Split(v, ",")
	sort.Strings(parts)
	return strings.Join(parts, ",")
}
