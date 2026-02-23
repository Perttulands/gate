package gates

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"polis/gate/internal/verdict"
)

// linterSpec describes one auto-detected linter.
type linterSpec struct {
	name string
	cmd  []string
}

// DetectLinters returns all applicable linters for the repo at dir.
func DetectLinters(dir string) []linterSpec {
	var linters []linterSpec

	// Go
	if fileExists(filepath.Join(dir, "go.mod")) {
		linters = append(linters, linterSpec{name: "go vet", cmd: []string{"go", "vet", "./..."}})
	}

	// Node/eslint
	if fileExists(filepath.Join(dir, "package.json")) {
		if hasESLint(dir) {
			linters = append(linters, linterSpec{name: "eslint", cmd: []string{"npx", "eslint", "."}})
		}
	}

	// Python
	pyFiles, err := filepath.Glob(filepath.Join(dir, "*.py"))
	pyDir := filepath.Join(dir, "src")
	hasPyDir := fileExists(pyDir)
	if (err == nil && len(pyFiles) > 0) || hasPyDir || fileExists(filepath.Join(dir, "pyproject.toml")) || fileExists(filepath.Join(dir, "setup.py")) {
		linters = append(linters, linterSpec{name: "ruff", cmd: []string{"ruff", "check", "."}})
	}

	// Shell
	shFiles, err := filepath.Glob(filepath.Join(dir, "*.sh"))
	if err == nil && len(shFiles) > 0 {
		args := []string{}
		for _, f := range shFiles {
			args = append(args, f)
		}
		linters = append(linters, linterSpec{name: "shellcheck", cmd: append([]string{"shellcheck"}, args...)})
	}

	return linters
}

// RunLint detects and runs all applicable linters for the repo at dir.
func RunLint(ctx context.Context, dir string, timeoutSec int) []verdict.GateResult {
	specs := DetectLinters(dir)
	if len(specs) == 0 {
		return []verdict.GateResult{{Name: "lint", Pass: true, Output: "no linters detected"}}
	}
	if timeoutSec <= 0 {
		timeoutSec = 60
	}

	var results []verdict.GateResult
	for _, s := range specs {
		spec := s
		r := verdict.TimedRun("lint:"+spec.name, func() (bool, string, error) {
			pass, output, err := runCmd(ctx, dir, timeoutSec, spec.cmd[0], spec.cmd[1:]...)
			return pass, output, err
		})
		results = append(results, r)
	}
	return results
}

// hasESLint checks if eslint is a devDependency or dependency in package.json.
func hasESLint(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}
	var pkg map[string]json.RawMessage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}
	for _, key := range []string{"devDependencies", "dependencies"} {
		if raw, ok := pkg[key]; ok {
			if strings.Contains(string(raw), "eslint") {
				return true
			}
		}
	}
	return false
}
