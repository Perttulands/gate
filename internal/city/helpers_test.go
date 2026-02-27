package city

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePolisPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"simple file", "polis.yaml", "polis.yaml", false},
		{"nested file", "config/polis.yaml", "config/polis.yaml", false},
		{"directory with slash", "memory/", "memory/", false},
		{"backslash converted", `config\polis.yaml`, "config/polis.yaml", false},
		{"whitespace trimmed", "  polis.yaml  ", "polis.yaml", false},
		{"dot segments cleaned", "config/./polis.yaml", "config/polis.yaml", false},
		{"double slash cleaned", "config//polis.yaml", "config/polis.yaml", false},
		{"glob preserved", "memory/**", "memory/**", false},
		{"empty string", "", "", true},
		{"whitespace only", "   ", "", true},
		{"absolute path", "/etc/passwd", "", true},
		{"current dir", ".", "", true},
		{"parent dir", "..", "", true},
		{"traversal prefix", "../secret", "", true},
		{"traversal nested", "a/../../secret", "", true},

		// --- additional edge cases ---
		{"deeply nested path", "a/b/c/d/e/f.txt", "a/b/c/d/e/f.txt", false},
		{"mixed separators", `a\b/c\d.txt`, "a/b/c/d.txt", false},
		{"dot in filename", "config/.hidden", "config/.hidden", false},
		{"multiple extensions", "file.tar.gz", "file.tar.gz", false},
		{"trailing dot cleaned", "config/./", "config/", false},
		{"redundant separators", "a///b///c", "a/b/c", false},
		{"glob question mark", "dir/file?.txt", "dir/file?.txt", false},
		{"glob bracket range", "dir/file[0-9].txt", "dir/file[0-9].txt", false},
		{"tilde not special", "~/config", "~/config", false},
		{"single char filename", "x", "x", false},
		{"dir marker preserved after clean", "a/./b/", "a/b/", false},
		{"tab in whitespace", "\t polis.yaml \t", "polis.yaml", false},
		{"absolute windows-style", `\absolute`, "", true},
		{"traversal mid path", "a/b/../../../secret", "", true},
		{"dot slash only", "./", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizePolisPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizePolisPath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("normalizePolisPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeHookPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"simple file", "polis.yaml", "polis.yaml", false},
		{"nested file", "config/polis.yaml", "config/polis.yaml", false},
		{"directory rejected", "memory/", "", true},
		{"glob rejected", "config/*.yaml", "", true},
		{"glob star star rejected", "config/**", "", true},
		{"glob question rejected", "file?.txt", "", true},
		{"glob bracket rejected", "file[0].txt", "", true},
		{"empty rejected", "", "", true},
		{"absolute rejected", "/etc/file", "", true},

		// --- additional edge cases ---
		{"deeply nested valid", "a/b/c/hook.sh", "a/b/c/hook.sh", false},
		{"dot file valid", ".secrets", ".secrets", false},
		{"backslash normalized", `hooks\run.sh`, "hooks/run.sh", false},
		{"whitespace trimmed", "  hook.yaml  ", "hook.yaml", false},
		{"double star glob rejected", "src/**/*.go", "", true},
		{"traversal rejected", "../hook.sh", "", true},
		{"current dir rejected", ".", "", true},
		{"whitespace only rejected", "   ", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeHookPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeHookPath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("normalizeHookPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasGlobMeta(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"plain.txt", false},
		{"path/to/file", false},
		{"*.txt", true},
		{"dir/**", true},
		{"file?.log", true},
		{"file[0].txt", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := hasGlobMeta(tt.input); got != tt.want {
				t.Fatalf("hasGlobMeta(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSynthesizePathFromPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{"single star", "*.txt", "sample.txt"},
		{"double star", "memory/**", "memory/sample"},
		{"question mark", "file?.log", "filex.log"},
		{"char class", "file[0-9].txt", "filex.txt"},
		{"no meta", "plain.txt", "plain.txt"},
		{"complex", "src/**/*.go", "src/sample/sample.go"},
		{"empty becomes sample", "", "sample"},
		{"trailing slash gets sample", "dir/*/", "dir/sample/sample"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := synthesizePathFromPattern(tt.pattern)
			if got != tt.want {
				t.Fatalf("synthesizePathFromPattern(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestIgnoreCandidate(t *testing.T) {
	tests := []struct {
		name  string
		entry string
		want  string
	}{
		{"plain file", "polis.yaml", "polis.yaml"},
		{"directory", "memory/", "memory/.gate-city-sample"},
		{"glob", "memory/**", "memory/sample"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ignoreCandidate(tt.entry)
			if got != tt.want {
				t.Fatalf("ignoreCandidate(%q) = %q, want %q", tt.entry, got, tt.want)
			}
		})
	}
}

func TestMatchGlobPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		rel     string
		want    bool
	}{
		{"exact match", "file.txt", "file.txt", true},
		{"exact no match", "file.txt", "other.txt", false},
		{"star matches file", "*.txt", "readme.txt", true},
		{"star no match ext", "*.txt", "readme.md", false},
		{"nested star", "src/*.go", "src/main.go", true},
		{"nested star no match depth", "src/*.go", "src/sub/main.go", false},
		{"double star any depth", "src/**/*.go", "src/main.go", true},
		{"double star deeper", "src/**/*.go", "src/pkg/main.go", true},
		{"double star deepest", "src/**/*.go", "src/a/b/c/main.go", true},
		{"double star no match ext", "src/**/*.go", "src/main.txt", false},
		{"double star alone", "**", "any/path/here", true},
		{"double star alone single", "**", "file.txt", true},
		{"question mark", "file?.txt", "file1.txt", true},
		{"question mark no match", "file?.txt", "file12.txt", false},
		{"directory glob", "memory/**", "memory/entry.txt", true},
		{"directory glob nested", "memory/**", "memory/sub/entry.txt", true},

		// --- additional edge cases ---
		{"bracket char class", "file[0-9].txt", "file3.txt", true},
		{"bracket char class no match", "file[0-9].txt", "fileA.txt", false},
		{"double star at start", "**/main.go", "src/pkg/main.go", true},
		{"double star at start shallow", "**/main.go", "main.go", true},
		{"double star at start no match", "**/main.go", "src/main.txt", false},
		{"multiple wildcards", "src/**/test/*.go", "src/pkg/test/foo.go", true},
		{"multiple wildcards deep", "src/**/test/*.go", "src/a/b/test/bar.go", true},
		{"multiple wildcards no match", "src/**/test/*.go", "src/pkg/prod/foo.go", false},
		{"empty rel no match", "*.txt", "", false},
		{"double star matches zero segments", "src/**", "src", true},
		{"star does not cross slash", "src/*", "src/a/b.go", false},
		{"exact nested match", "a/b/c.txt", "a/b/c.txt", true},
		{"exact nested no match", "a/b/c.txt", "a/b/d.txt", false},
		{"double star zero segments", "a/**/b.txt", "a/b.txt", true},
		{"double star one segment", "a/**/b.txt", "a/x/b.txt", true},
		{"double star many segments", "a/**/b.txt", "a/x/y/z/b.txt", true},
		{"pattern longer than rel", "a/b/c/d", "a/b", false},
		{"rel longer than pattern", "a/b", "a/b/c/d", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchGlobPattern(tt.pattern, tt.rel)
			if got != tt.want {
				t.Fatalf("matchGlobPattern(%q, %q) = %v, want %v", tt.pattern, tt.rel, got, tt.want)
			}
		})
	}
}

func TestSplitSegments(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"a/b/c", 3},
		{"single", 1},
		{"a//b", 2},
		{"./a", 1},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitSegments(tt.input)
			if len(got) != tt.want {
				t.Fatalf("splitSegments(%q) = %v (len %d), want len %d", tt.input, got, len(got), tt.want)
			}
		})
	}
}

func TestSummarize(t *testing.T) {
	results := []CheckResult{
		{Status: StatusPass},
		{Status: StatusPass},
		{Status: StatusFail},
		{Status: StatusSkip},
	}
	s := summarize(results)
	if s.Pass != 2 || s.Fail != 1 || s.Skip != 1 {
		t.Fatalf("summarize = %+v, want pass=2 fail=1 skip=1", s)
	}

	empty := summarize(nil)
	if empty.Pass != 0 || empty.Fail != 0 || empty.Skip != 0 {
		t.Fatalf("summarize(nil) = %+v, want all zeros", empty)
	}
}

func TestModeKind(t *testing.T) {
	tests := []struct {
		name string
		mode os.FileMode
		want string
	}{
		{"regular", 0, "file"},
		{"directory", os.ModeDir, "directory"},
		{"symlink", os.ModeSymlink, "symlink"},
		{"other", os.ModeNamedPipe, "non-regular"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modeKind(tt.mode)
			if got != tt.want {
				t.Fatalf("modeKind(%v) = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}

func TestTrimOutput(t *testing.T) {
	tests := []struct {
		name string
		out  string
		err  error
		want string
	}{
		{"single line", "hello\n", nil, "hello"},
		{"multi line under limit", "a\nb\nc\n", nil, "a | b | c"},
		{"truncated to 4 lines", "1\n2\n3\n4\n5\n6\n", nil, "1 | 2 | 3 | 4"},
		{"empty with error", "", os.ErrNotExist, "file does not exist"},
		{"empty no error", "", nil, "unknown error"},
		{"whitespace only with error", "   \n  ", os.ErrPermission, "permission denied"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimOutput(tt.out, tt.err)
			if got != tt.want {
				t.Fatalf("trimOutput(%q, %v) = %q, want %q", tt.out, tt.err, got, tt.want)
			}
		})
	}
}

func TestCheckHooks(t *testing.T) {
	t.Run("no hooks declared", func(t *testing.T) {
		cfg := Config{PolisFiles: []string{"polis.yaml"}}
		status, detail := checkHooks(cfg, "")
		if status != StatusPass {
			t.Fatalf("expected pass, got %s: %s", status, detail)
		}
		if detail != "no hooks declared" {
			t.Fatalf("expected 'no hooks declared', got %q", detail)
		}
	})

	t.Run("defaults fallback always valid", func(t *testing.T) {
		cfg := Config{
			PolisFiles: []string{"polis.yaml"},
			Hooks:      []Hook{{File: "polis.yaml", Fallback: "defaults"}},
		}
		status, detail := checkHooks(cfg, "")
		if status != StatusPass {
			t.Fatalf("expected pass, got %s: %s", status, detail)
		}
		if !strings.Contains(detail, "1 hooks sound") {
			t.Fatalf("expected '1 hooks sound', got %q", detail)
		}
	})

	t.Run("valid env fallback", func(t *testing.T) {
		cfg := Config{
			PolisFiles: []string{".secrets"},
			Hooks:      []Hook{{File: ".secrets", Fallback: "env:POLIS_API_KEY"}},
		}
		status, detail := checkHooks(cfg, "")
		if status != StatusPass {
			t.Fatalf("expected pass, got %s: %s", status, detail)
		}
	})

	t.Run("valid env fallback underscore prefix", func(t *testing.T) {
		cfg := Config{
			PolisFiles: []string{".secrets"},
			Hooks:      []Hook{{File: ".secrets", Fallback: "env:_PRIVATE_VAR"}},
		}
		status, detail := checkHooks(cfg, "")
		if status != StatusPass {
			t.Fatalf("expected pass, got %s: %s", status, detail)
		}
	})

	t.Run("invalid env fallback lowercase", func(t *testing.T) {
		cfg := Config{
			PolisFiles: []string{".secrets"},
			Hooks:      []Hook{{File: ".secrets", Fallback: "env:lower_case"}},
		}
		status, detail := checkHooks(cfg, "")
		if status != StatusFail {
			t.Fatalf("expected fail, got %s: %s", status, detail)
		}
		if !strings.Contains(detail, "invalid env fallback") {
			t.Fatalf("expected invalid env fallback message, got %q", detail)
		}
	})

	t.Run("invalid env fallback empty name", func(t *testing.T) {
		cfg := Config{
			PolisFiles: []string{".secrets"},
			Hooks:      []Hook{{File: ".secrets", Fallback: "env:"}},
		}
		status, detail := checkHooks(cfg, "")
		if status != StatusFail {
			t.Fatalf("expected fail, got %s: %s", status, detail)
		}
		if !strings.Contains(detail, "invalid env fallback") {
			t.Fatalf("expected invalid env fallback message, got %q", detail)
		}
	})

	t.Run("invalid fallback string rejected", func(t *testing.T) {
		cfg := Config{
			PolisFiles: []string{"polis.yaml"},
			Hooks:      []Hook{{File: "polis.yaml", Fallback: "something-wrong"}},
		}
		status, detail := checkHooks(cfg, "")
		if status != StatusFail {
			t.Fatalf("expected fail, got %s: %s", status, detail)
		}
		if !strings.Contains(detail, "invalid fallback") {
			t.Fatalf("expected invalid fallback message, got %q", detail)
		}
	})

	t.Run("empty fallback rejected", func(t *testing.T) {
		cfg := Config{
			PolisFiles: []string{"polis.yaml"},
			Hooks:      []Hook{{File: "polis.yaml", Fallback: ""}},
		}
		status, detail := checkHooks(cfg, "")
		if status != StatusFail {
			t.Fatalf("expected fail, got %s: %s", status, detail)
		}
		if !strings.Contains(detail, "invalid fallback") {
			t.Fatalf("expected invalid fallback message, got %q", detail)
		}
	})

	t.Run("hook file not in polis_files", func(t *testing.T) {
		cfg := Config{
			PolisFiles: []string{"other.yaml"},
			Hooks:      []Hook{{File: "missing.yaml", Fallback: "defaults"}},
		}
		status, detail := checkHooks(cfg, "")
		if status != StatusFail {
			t.Fatalf("expected fail, got %s: %s", status, detail)
		}
		if !strings.Contains(detail, "not listed in polis_files") {
			t.Fatalf("expected 'not listed in polis_files', got %q", detail)
		}
	})

	t.Run("fallback fail without install-at", func(t *testing.T) {
		cfg := Config{
			PolisFiles: []string{".secrets"},
			Hooks:      []Hook{{File: ".secrets", Fallback: "fail"}},
		}
		status, detail := checkHooks(cfg, "")
		if status != StatusFail {
			t.Fatalf("expected fail, got %s: %s", status, detail)
		}
		if !strings.Contains(detail, "requires --install-at") {
			t.Fatalf("expected install-at guidance, got %q", detail)
		}
	})

	t.Run("fallback fail with missing file", func(t *testing.T) {
		install := t.TempDir()
		cfg := Config{
			PolisFiles: []string{".secrets"},
			Hooks:      []Hook{{File: ".secrets", Fallback: "fail"}},
		}
		status, detail := checkHooks(cfg, install)
		if status != StatusFail {
			t.Fatalf("expected fail, got %s: %s", status, detail)
		}
		if !strings.Contains(detail, "file missing at install path") {
			t.Fatalf("expected missing file message, got %q", detail)
		}
	})

	t.Run("fallback fail with file present", func(t *testing.T) {
		install := t.TempDir()
		if err := os.WriteFile(filepath.Join(install, ".secrets"), []byte("ok"), 0o644); err != nil {
			t.Fatal(err)
		}
		cfg := Config{
			PolisFiles: []string{".secrets"},
			Hooks:      []Hook{{File: ".secrets", Fallback: "fail"}},
		}
		status, detail := checkHooks(cfg, install)
		if status != StatusPass {
			t.Fatalf("expected pass, got %s: %s", status, detail)
		}
	})

	t.Run("fallback fail with symlink rejected", func(t *testing.T) {
		install := t.TempDir()
		real := filepath.Join(install, "real.txt")
		if err := os.WriteFile(real, []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, filepath.Join(install, ".secrets")); err != nil {
			t.Fatal(err)
		}
		cfg := Config{
			PolisFiles: []string{".secrets"},
			Hooks:      []Hook{{File: ".secrets", Fallback: "fail"}},
		}
		status, detail := checkHooks(cfg, install)
		if status != StatusFail {
			t.Fatalf("expected fail, got %s: %s", status, detail)
		}
		if !strings.Contains(detail, "symlink") {
			t.Fatalf("expected symlink message, got %q", detail)
		}
	})

	t.Run("multiple hooks mixed results", func(t *testing.T) {
		cfg := Config{
			PolisFiles: []string{"polis.yaml", ".secrets"},
			Hooks: []Hook{
				{File: "polis.yaml", Fallback: "defaults"},
				{File: ".secrets", Fallback: "bogus"},
			},
		}
		status, detail := checkHooks(cfg, "")
		if status != StatusFail {
			t.Fatalf("expected fail, got %s: %s", status, detail)
		}
		if !strings.Contains(detail, "invalid fallback") {
			t.Fatalf("expected invalid fallback in detail, got %q", detail)
		}
	})

	t.Run("multiple hooks all valid", func(t *testing.T) {
		cfg := Config{
			PolisFiles: []string{"polis.yaml", ".secrets"},
			Hooks: []Hook{
				{File: "polis.yaml", Fallback: "defaults"},
				{File: ".secrets", Fallback: "env:API_KEY"},
			},
		}
		status, detail := checkHooks(cfg, "")
		if status != StatusPass {
			t.Fatalf("expected pass, got %s: %s", status, detail)
		}
		if !strings.Contains(detail, "2 hooks sound") {
			t.Fatalf("expected '2 hooks sound', got %q", detail)
		}
	})

	t.Run("polis_files with dir suffix matches hook", func(t *testing.T) {
		cfg := Config{
			PolisFiles: []string{"memory/"},
			Hooks:      []Hook{{File: "memory", Fallback: "defaults"}},
		}
		status, _ := checkHooks(cfg, "")
		if status != StatusPass {
			t.Fatalf("expected pass: dir-suffix polis_files should match hook file without slash")
		}
	})
}
