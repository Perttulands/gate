package city

import (
	"os"
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
