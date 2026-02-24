package city

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var execCommand = exec.Command

func TestRun_InvalidContract_MissingSchemaVersion(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, ".gitignore", "polis.yaml\n")
	writeFile(t, repo, "city.toml", `
[city]
polis_files = ["polis.yaml"]
standalone_check = ""
`)
	initGitRepo(t, repo)

	v := Run(context.Background(), repo, Options{SkipStandalone: true})
	if v.ExitCode != ExitInvalid {
		t.Fatalf("expected invalid exit code %d, got %d", ExitInvalid, v.ExitCode)
	}
	if len(v.Checks) != 1 || v.Checks[0].Name != "contract" {
		t.Fatalf("expected single contract check, got %+v", v.Checks)
	}
}

func TestRun_BoundaryUsesGitSemantics(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, ".gitignore", "memory/*\n!memory/public.txt\n")
	writeFile(t, repo, "city.toml", `
[city]
schema_version = 1
polis_files = ["memory/public.txt"]
standalone_check = ""
`)
	initGitRepo(t, repo)

	v := Run(context.Background(), repo, Options{SkipStandalone: true})
	boundary := findCheck(t, v, "boundary")
	if boundary.Status != StatusFail {
		t.Fatalf("expected boundary fail, got %+v", boundary)
	}
	if !strings.Contains(boundary.Detail, "memory/public.txt") {
		t.Fatalf("expected missing path in detail, got %q", boundary.Detail)
	}
}

func TestRun_WarnWhenOnlySkips(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, ".gitignore", "polis.yaml\n")
	writeFile(t, repo, "city.toml", `
[city]
schema_version = 1
polis_files = ["polis.yaml"]
standalone_check = ""
`)
	initGitRepo(t, repo)

	v := Run(context.Background(), repo, Options{})
	if v.ExitCode != ExitWarn {
		t.Fatalf("expected warn exit %d, got %d (%+v)", ExitWarn, v.ExitCode, v)
	}
	if v.Status != "warn" {
		t.Fatalf("expected warn status, got %q", v.Status)
	}
}

func TestRun_PassAllChecks(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, ".gitignore", "polis.yaml\n.secrets\nmemory/\n")
	writeFile(t, repo, "city.toml", `
[city]
schema_version = 1
polis_files = ["polis.yaml", ".secrets", "memory/", "memory/**"]
standalone_check = "true"

[[hook]]
file = "polis.yaml"
fallback = "defaults"

[[hook]]
file = ".secrets"
fallback = "env:POLIS_API_KEY"
`)
	initGitRepo(t, repo)

	install := t.TempDir()
	writeFile(t, install, "polis.yaml", "city: true\n")
	writeFile(t, install, ".secrets", "token=abc\n")
	mkdirAll(t, filepath.Join(install, "memory"))
	writeFile(t, install, "memory/entry.txt", "ok\n")

	v := Run(context.Background(), repo, Options{
		InstallAt:         install,
		StandaloneTimeout: 2 * time.Second,
	})
	if v.ExitCode != ExitPass {
		t.Fatalf("expected pass exit %d, got %d: %+v", ExitPass, v.ExitCode, v)
	}
	if !v.Pass || v.Status != "pass" {
		t.Fatalf("expected pass status, got %+v", v)
	}
}

func TestRun_HooksFailWhenFallbackFailWithoutInstallPath(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, ".gitignore", ".secrets\n")
	writeFile(t, repo, "city.toml", `
[city]
schema_version = 1
polis_files = [".secrets"]
standalone_check = ""

[[hook]]
file = ".secrets"
fallback = "fail"
`)
	initGitRepo(t, repo)

	v := Run(context.Background(), repo, Options{SkipStandalone: true})
	hooks := findCheck(t, v, "config-hooks")
	if hooks.Status != StatusFail {
		t.Fatalf("expected hooks failure, got %+v", hooks)
	}
	if !strings.Contains(hooks.Detail, "requires --install-at") {
		t.Fatalf("expected install-at guidance, got %q", hooks.Detail)
	}
}

func TestCheckSplit_FailsOnTypeMismatchAndSymlink(t *testing.T) {
	install := t.TempDir()
	mkdirAll(t, filepath.Join(install, "polis.yaml"))
	mkdirAll(t, filepath.Join(install, "memory"))
	writeFile(t, install, "memory/file.txt", "x\n")
	if err := os.Symlink(filepath.Join(install, "memory", "file.txt"), filepath.Join(install, ".secrets")); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	status, detail := checkSplit([]string{"polis.yaml", ".secrets", "memory/"}, install)
	if status != StatusFail {
		t.Fatalf("expected split failure, got %s (%s)", status, detail)
	}
	if !strings.Contains(detail, "expected file") {
		t.Fatalf("expected file mismatch detail, got %q", detail)
	}
	if !strings.Contains(detail, "symlink") {
		t.Fatalf("expected symlink detail, got %q", detail)
	}
}

func TestRun_StandaloneTimeoutFails(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, ".gitignore", "polis.yaml\n")
	writeFile(t, repo, "city.toml", `
[city]
schema_version = 1
polis_files = ["polis.yaml"]
standalone_check = "sleep 1"
`)
	initGitRepo(t, repo)

	install := t.TempDir()
	writeFile(t, install, "polis.yaml", "ok\n")

	v := Run(context.Background(), repo, Options{
		InstallAt:         install,
		StandaloneTimeout: 10 * time.Millisecond,
	})
	standalone := findCheck(t, v, "standalone")
	if standalone.Status != StatusFail {
		t.Fatalf("expected standalone timeout failure, got %+v", standalone)
	}
	if !strings.Contains(standalone.Detail, "timed out") {
		t.Fatalf("expected timeout detail, got %q", standalone.Detail)
	}
}

func findCheck(t *testing.T, v Verdict, name string) CheckResult {
	t.Helper()
	for _, c := range v.Checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("check %q not found in %+v", name, v.Checks)
	return CheckResult{}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	mustRun(t, dir, "git", "init")
	mustRun(t, dir, "git", "config", "user.email", "gate-tests@example.com")
	mustRun(t, dir, "git", "config", "user.name", "gate-tests")
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "init")
}

func mustRun(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := execCommand(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v (%s)", name, args, err, string(out))
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", target, err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", target, err)
	}
}

func mkdirAll(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}
