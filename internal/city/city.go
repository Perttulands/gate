package city

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

const (
	// ExitPass means all checks passed.
	ExitPass = 0
	// ExitFail means one or more checks failed.
	ExitFail = 1
	// ExitWarn means no failures, but one or more checks were skipped.
	ExitWarn = 2
	// ExitInvalid means city contract/input is invalid.
	ExitInvalid = 3
)

const (
	StatusPass = "pass"
	StatusFail = "fail"
	StatusSkip = "skip"
)

const defaultStandaloneTimeout = 120 * time.Second

var envFallbackRe = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

type rawCityFile struct {
	City rawCityConfig `toml:"city"`
	Hook []Hook        `toml:"hook"`
}

type rawCityConfig struct {
	SchemaVersion   *int     `toml:"schema_version"`
	PolisFiles      []string `toml:"polis_files"`
	StandaloneCheck string   `toml:"standalone_check"`
}

// Hook is a declared config hook in city.toml.
type Hook struct {
	File     string `toml:"file"`
	Fallback string `toml:"fallback"`
}

// Config is validated city.toml data.
type Config struct {
	SchemaVersion   int
	PolisFiles      []string
	StandaloneCheck string
	Hooks           []Hook
}

// Options controls gate city execution.
type Options struct {
	InstallAt         string
	SkipStandalone    bool
	StandaloneTimeout time.Duration
}

// CheckResult is one city check outcome.
type CheckResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Detail     string `json:"detail"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// Summary holds counts by status.
type Summary struct {
	Pass int `json:"pass"`
	Fail int `json:"fail"`
	Skip int `json:"skip"`
}

// Verdict is the final gate city result.
type Verdict struct {
	Pass     bool          `json:"pass"`
	Status   string        `json:"status"`
	Repo     string        `json:"repo"`
	Checks   []CheckResult `json:"checks"`
	Summary  Summary       `json:"summary"`
	ExitCode int           `json:"exit_code"`
	Bead     string        `json:"bead,omitempty"`
}

// ContractError marks malformed city contract/input.
type ContractError struct {
	Msg string
}

func (e ContractError) Error() string {
	return e.Msg
}

// Run executes all four city checks.
func Run(ctx context.Context, repoPath string, opts Options) Verdict {
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return invalidVerdict(repoPath, fmt.Sprintf("invalid repo path: %v", err))
	}
	repoName := filepath.Base(absRepo)

	if opts.StandaloneTimeout <= 0 {
		opts.StandaloneTimeout = defaultStandaloneTimeout
	}

	if err := ensureGitRepo(absRepo); err != nil {
		return invalidVerdict(repoName, fmt.Sprintf("invalid repo input: %v", err))
	}

	cfg, err := loadConfig(absRepo)
	if err != nil {
		return invalidVerdict(repoName, err.Error())
	}

	results := make([]CheckResult, 0, 4)
	results = append(results, timedCheck("boundary", func() (string, string) {
		return checkBoundary(absRepo, cfg.PolisFiles)
	}))
	results = append(results, timedCheck("standalone", func() (string, string) {
		return checkStandalone(ctx, absRepo, cfg, opts)
	}))
	results = append(results, timedCheck("config-hooks", func() (string, string) {
		return checkHooks(cfg, opts.InstallAt)
	}))
	results = append(results, timedCheck("split", func() (string, string) {
		return checkSplit(cfg.PolisFiles, opts.InstallAt)
	}))

	summary := summarize(results)
	v := Verdict{
		Repo:    repoName,
		Checks:  results,
		Summary: summary,
	}

	switch {
	case summary.Fail > 0:
		v.Status = "fail"
		v.Pass = false
		v.ExitCode = ExitFail
	case summary.Skip > 0:
		v.Status = "warn"
		v.Pass = false
		v.ExitCode = ExitWarn
	default:
		v.Status = "pass"
		v.Pass = true
		v.ExitCode = ExitPass
	}
	return v
}

func invalidVerdict(repo, detail string) Verdict {
	return Verdict{
		Pass:     false,
		Status:   "fail",
		Repo:     repo,
		Checks:   []CheckResult{{Name: "contract", Status: StatusFail, Detail: detail}},
		Summary:  Summary{Fail: 1},
		ExitCode: ExitInvalid,
	}
}

func timedCheck(name string, fn func() (string, string)) CheckResult {
	start := time.Now()
	status, detail := fn()
	return CheckResult{
		Name:       name,
		Status:     status,
		Detail:     detail,
		DurationMs: time.Since(start).Milliseconds(),
	}
}

func summarize(results []CheckResult) Summary {
	var s Summary
	for _, r := range results {
		switch r.Status {
		case StatusPass:
			s.Pass++
		case StatusFail:
			s.Fail++
		case StatusSkip:
			s.Skip++
		}
	}
	return s
}

func ensureGitRepo(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("not a git work tree: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func loadConfig(repoPath string) (Config, error) {
	cfgPath := filepath.Join(repoPath, "city.toml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return Config{}, ContractError{Msg: fmt.Sprintf("invalid city.toml: %v", err)}
	}

	var raw rawCityFile
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Config{}, ContractError{Msg: fmt.Sprintf("invalid city.toml TOML: %v", err)}
	}

	if raw.City.SchemaVersion == nil {
		return Config{}, ContractError{Msg: "invalid city.toml: [city].schema_version is required"}
	}
	if *raw.City.SchemaVersion != 1 {
		return Config{}, ContractError{Msg: fmt.Sprintf("invalid city.toml: unsupported schema_version %d (expected 1)", *raw.City.SchemaVersion)}
	}

	polisFiles := make([]string, 0, len(raw.City.PolisFiles))
	for _, entry := range raw.City.PolisFiles {
		norm, err := normalizePolisPath(entry)
		if err != nil {
			return Config{}, ContractError{Msg: fmt.Sprintf("invalid city.toml polis_files entry %q: %v", entry, err)}
		}
		polisFiles = append(polisFiles, norm)
	}

	hooks := make([]Hook, 0, len(raw.Hook))
	for _, h := range raw.Hook {
		file, err := normalizeHookPath(h.File)
		if err != nil {
			return Config{}, ContractError{Msg: fmt.Sprintf("invalid city.toml hook.file %q: %v", h.File, err)}
		}
		hooks = append(hooks, Hook{
			File:     file,
			Fallback: strings.TrimSpace(h.Fallback),
		})
	}

	return Config{
		SchemaVersion:   *raw.City.SchemaVersion,
		PolisFiles:      polisFiles,
		StandaloneCheck: strings.TrimSpace(raw.City.StandaloneCheck),
		Hooks:           hooks,
	}, nil
}

func normalizePolisPath(p string) (string, error) {
	v := strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	if v == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if path.IsAbs(v) {
		return "", fmt.Errorf("path must be relative")
	}
	keepDirMarker := strings.HasSuffix(v, "/")
	clean := path.Clean(v)
	if clean == "." {
		return "", fmt.Errorf("path cannot be current directory")
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path traversal (..) is not allowed")
	}
	if keepDirMarker && clean != "/" {
		clean += "/"
	}
	return clean, nil
}

func normalizeHookPath(p string) (string, error) {
	clean, err := normalizePolisPath(p)
	if err != nil {
		return "", fmt.Errorf("invalid hook path: %w", err)
	}
	if strings.HasSuffix(clean, "/") {
		return "", fmt.Errorf("hook file cannot be a directory path")
	}
	if hasGlobMeta(clean) {
		return "", fmt.Errorf("hook file cannot include glob meta")
	}
	return clean, nil
}

func checkBoundary(repoPath string, polisFiles []string) (string, string) {
	if len(polisFiles) == 0 {
		return StatusPass, "no polis_files declared"
	}

	var missing []string
	for _, entry := range polisFiles {
		candidate := ignoreCandidate(entry)
		ignored, err := gitIgnored(repoPath, candidate)
		if err != nil {
			return StatusFail, fmt.Sprintf("git check-ignore failed for %q: %v", entry, err)
		}
		if !ignored {
			missing = append(missing, entry)
		}
	}

	if len(missing) > 0 {
		return StatusFail, fmt.Sprintf("not ignored by Git semantics: %s", strings.Join(missing, ", "))
	}
	return StatusPass, fmt.Sprintf("%d polis_files all ignored by Git semantics", len(polisFiles))
}

func ignoreCandidate(entry string) string {
	if hasGlobMeta(entry) {
		return synthesizePathFromPattern(entry)
	}
	if strings.HasSuffix(entry, "/") {
		return path.Join(strings.TrimSuffix(entry, "/"), ".gate-city-sample")
	}
	return entry
}

func synthesizePathFromPattern(pattern string) string {
	var b strings.Builder
	inClass := false
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		if inClass {
			if ch == ']' {
				inClass = false
				b.WriteByte('x')
			}
			continue
		}
		switch ch {
		case '[':
			inClass = true
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i++
			}
			b.WriteString("sample")
		case '?':
			b.WriteByte('x')
		default:
			b.WriteByte(ch)
		}
	}
	v := b.String()
	if v == "" {
		v = "sample"
	}
	if strings.HasSuffix(v, "/") {
		v += "sample"
	}
	return path.Clean(v)
}

func gitIgnored(repoPath, relPath string) (bool, error) {
	cmd := exec.Command("git", "-C", repoPath, "check-ignore", "-q", "--no-index", relPath)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("exit %d", exitErr.ExitCode())
	}
	return false, err
}

func checkStandalone(ctx context.Context, repoPath string, cfg Config, opts Options) (string, string) {
	if opts.SkipStandalone {
		return StatusSkip, "skipped by --skip-standalone"
	}
	if strings.TrimSpace(cfg.StandaloneCheck) == "" {
		return StatusSkip, "standalone_check empty in city.toml"
	}

	tmpDir, err := os.MkdirTemp("", "gate-city-*")
	if err != nil {
		return StatusFail, fmt.Sprintf("failed to prepare temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneDir := filepath.Join(tmpDir, "repo")
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--quiet", "--depth", "1", repoPath, cloneDir)
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		return StatusFail, fmt.Sprintf("clone failed: %s", trimOutput(string(out), err))
	}

	cmdCtx, cancel := context.WithTimeout(ctx, opts.StandaloneTimeout)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "bash", "-lc", cfg.StandaloneCheck)
	cmd.Dir = cloneDir
	cmd.Env = isolatedEnv()
	out, err := cmd.CombinedOutput()
	if cmdCtx.Err() == context.DeadlineExceeded {
		return StatusFail, fmt.Sprintf("standalone_check timed out after %s", opts.StandaloneTimeout)
	}
	if err != nil {
		return StatusFail, fmt.Sprintf("standalone_check failed: %s", trimOutput(string(out), err))
	}
	return StatusPass, "standalone_check exited 0"
}

func isolatedEnv() []string {
	keys := []string{"PATH", "HOME", "TMPDIR", "LANG", "LC_ALL", "TERM"}
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		val, ok := os.LookupEnv(key)
		if ok && val != "" {
			env = append(env, key+"="+val)
		}
	}
	return env
}

func trimOutput(out string, err error) string {
	s := strings.TrimSpace(out)
	if s == "" {
		if err == nil {
			return "unknown error"
		}
		return err.Error()
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 4 {
		lines = lines[:4]
	}
	return strings.Join(lines, " | ")
}

func checkHooks(cfg Config, installAt string) (string, string) {
	if len(cfg.Hooks) == 0 {
		return StatusPass, "no hooks declared"
	}

	polisSet := make(map[string]bool, len(cfg.PolisFiles))
	for _, f := range cfg.PolisFiles {
		polisSet[strings.TrimSuffix(f, "/")] = true
	}

	var problems []string
	for _, h := range cfg.Hooks {
		if !polisSet[h.File] {
			problems = append(problems, fmt.Sprintf("%s not listed in polis_files", h.File))
		}

		switch {
		case h.Fallback == "defaults":
		case h.Fallback == "fail":
			if installAt == "" {
				problems = append(problems, fmt.Sprintf("%s fallback=fail requires --install-at", h.File))
				continue
			}
			target := filepath.Join(installAt, filepath.FromSlash(h.File))
			info, err := os.Lstat(target)
			if err != nil {
				log.Printf("checkHooks: lstat failed for %s: %v", target, err)
				problems = append(problems, fmt.Sprintf("%s fallback=fail but file missing at install path", h.File))
				continue
			}
			if info.Mode()&os.ModeSymlink != 0 {
				problems = append(problems, fmt.Sprintf("%s fallback=fail but install path is symlink", h.File))
			}
		case strings.HasPrefix(h.Fallback, "env:"):
			envVar := strings.TrimPrefix(h.Fallback, "env:")
			if !envFallbackRe.MatchString(envVar) {
				problems = append(problems, fmt.Sprintf("%s has invalid env fallback %q", h.File, h.Fallback))
			}
		default:
			problems = append(problems, fmt.Sprintf("%s has invalid fallback %q", h.File, h.Fallback))
		}
	}

	if len(problems) > 0 {
		return StatusFail, strings.Join(problems, "; ")
	}
	return StatusPass, fmt.Sprintf("%d hooks sound", len(cfg.Hooks))
}

func checkSplit(polisFiles []string, installAt string) (string, string) {
	if installAt == "" {
		return StatusSkip, "skipped: --install-at not provided"
	}

	var missing []string
	for _, entry := range polisFiles {
		switch {
		case hasGlobMeta(entry):
			ok, err := hasGlobMatch(installAt, entry)
			if err != nil {
				log.Printf("checkSplit: glob match failed for %s: %v", entry, err)
				missing = append(missing, fmt.Sprintf("%s check failed: %v", entry, err))
				continue
			}
			if !ok {
				missing = append(missing, fmt.Sprintf("%s missing (glob no matches)", entry))
			}
		case strings.HasSuffix(entry, "/"):
			rel := strings.TrimSuffix(entry, "/")
			target := filepath.Join(installAt, filepath.FromSlash(rel))
			info, err := os.Lstat(target)
			if err != nil {
				log.Printf("checkSplit: missing directory %s: %v", target, err)
				missing = append(missing, fmt.Sprintf("%s missing at %s", entry, target))
				continue
			}
			if info.Mode()&os.ModeSymlink != 0 {
				missing = append(missing, fmt.Sprintf("%s expected directory but found symlink at %s", entry, target))
				continue
			}
			if !info.IsDir() {
				missing = append(missing, fmt.Sprintf("%s expected directory but found %s at %s", entry, modeKind(info.Mode()), target))
			}
		default:
			target := filepath.Join(installAt, filepath.FromSlash(entry))
			info, err := os.Lstat(target)
			if err != nil {
				log.Printf("checkSplit: missing file %s: %v", target, err)
				missing = append(missing, fmt.Sprintf("%s missing at %s", entry, target))
				continue
			}
			if info.Mode()&os.ModeSymlink != 0 {
				missing = append(missing, fmt.Sprintf("%s expected file but found symlink at %s", entry, target))
				continue
			}
			if !info.Mode().IsRegular() {
				missing = append(missing, fmt.Sprintf("%s expected file but found %s at %s", entry, modeKind(info.Mode()), target))
			}
		}
	}

	if len(missing) > 0 {
		return StatusFail, strings.Join(missing, "; ")
	}
	return StatusPass, fmt.Sprintf("%d polis files present at install path", len(polisFiles))
}

func modeKind(m fs.FileMode) string {
	kind := "non-regular"
	if m.IsDir() {
		kind = "directory"
	} else if m.IsRegular() {
		kind = "file"
	} else if m&os.ModeSymlink != 0 {
		kind = "symlink"
	}
	return kind
}

func hasGlobMatch(root, pattern string) (bool, error) {
	const matchFound = "match-found"
	errFound := errors.New(matchFound)

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", p, err)
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return fmt.Errorf("rel path %s: %w", p, err)
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if matchGlobPattern(pattern, rel) {
			return errFound
		}
		return nil
	})
	if err == nil {
		return false, nil
	}
	if errors.Is(err, errFound) {
		return true, nil
	}
	return false, err
}

func hasGlobMeta(v string) bool {
	return strings.ContainsAny(v, "*?[")
}

func matchGlobPattern(pattern, rel string) bool {
	pSeg := splitSegments(path.Clean(pattern))
	rSeg := splitSegments(path.Clean(rel))
	return matchSegments(pSeg, rSeg)
}

func splitSegments(v string) []string {
	parts := strings.Split(v, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" && p != "." {
			out = append(out, p)
		}
	}
	return out
}

func matchSegments(pattern, value []string) bool {
	var rec func(i, j int) bool
	rec = func(i, j int) bool {
		if i == len(pattern) && j == len(value) {
			return true
		}
		if i == len(pattern) {
			return false
		}
		if pattern[i] == "**" {
			if rec(i+1, j) {
				return true
			}
			if j < len(value) {
				return rec(i, j+1)
			}
			return false
		}
		if j >= len(value) {
			return false
		}
		ok, err := path.Match(pattern[i], value[j])
		if err != nil || !ok {
			return false
		}
		return rec(i+1, j+1)
	}
	return rec(0, 0)
}
