package main

import (
	"os"
	"testing"
)

func TestValidateFilterValue(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid simple", "--repo", "gate", "gate", false},
		{"valid with dots", "--repo", "my.repo", "my.repo", false},
		{"valid with dash", "--citizen", "john-doe", "john-doe", false},
		{"valid with underscore", "--repo", "my_repo", "my_repo", false},
		{"trims whitespace", "--repo", "  gate  ", "gate", false},
		{"empty after trim", "--repo", "   ", "", true},
		{"empty string", "--repo", "", "", true},
		{"invalid chars slash", "--repo", "foo/bar", "", true},
		{"invalid chars space", "--repo", "foo bar", "", true},
		{"invalid chars colon", "--citizen", "foo:bar", "", true},
		{"invalid chars at", "--repo", "user@host", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateFilterValue(tt.flag, tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateFilterValue(%q, %q) error = %v, wantErr %v", tt.flag, tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("validateFilterValue(%q, %q) = %q, want %q", tt.flag, tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveCitizen(t *testing.T) {
	t.Run("explicit value wins", func(t *testing.T) {
		got := resolveCitizen("alice")
		if got != "alice" {
			t.Fatalf("got %q, want %q", got, "alice")
		}
	})

	t.Run("trims explicit whitespace", func(t *testing.T) {
		got := resolveCitizen("  bob  ")
		if got != "bob" {
			t.Fatalf("got %q, want %q", got, "bob")
		}
	})

	t.Run("env var when explicit empty", func(t *testing.T) {
		t.Setenv("POLIS_CITIZEN", "env-user")
		got := resolveCitizen("")
		if got != "env-user" {
			t.Fatalf("got %q, want %q", got, "env-user")
		}
	})

	t.Run("env var trimmed", func(t *testing.T) {
		t.Setenv("POLIS_CITIZEN", "  spaced  ")
		got := resolveCitizen("")
		if got != "spaced" {
			t.Fatalf("got %q, want %q", got, "spaced")
		}
	})

	t.Run("empty env falls through", func(t *testing.T) {
		t.Setenv("POLIS_CITIZEN", "")
		// Will fall through to git user.name or "unknown"
		got := resolveCitizen("")
		// We can't predict git config, but it shouldn't be empty
		if got == "" {
			t.Fatal("resolveCitizen should never return empty string")
		}
	})
}

func TestRun_Help(t *testing.T) {
	// Redirect stdout to discard help output
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = old }()

	tests := []struct {
		name string
		args []string
	}{
		{"no args", nil},
		{"help", []string{"help"}},
		{"--help", []string{"--help"}},
		{"-h", []string{"-h"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := run(nil, tt.args)
			if code != 0 {
				t.Fatalf("run(%v) = %d, want 0", tt.args, code)
			}
		})
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = old }()

	oldErr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() { os.Stderr = oldErr }()

	code := run(nil, []string{"bogus"})
	if code != 1 {
		t.Fatalf("run(bogus) = %d, want 1", code)
	}
}

func TestRunCheck_MissingRepo(t *testing.T) {
	oldErr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() { os.Stderr = oldErr }()

	code := runCheck(nil, []string{})
	if code != 1 {
		t.Fatalf("runCheck with no repo = %d, want 1", code)
	}
}

func TestRunCheck_FlagErrors(t *testing.T) {
	oldErr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() { os.Stderr = oldErr }()

	tests := []struct {
		name string
		args []string
	}{
		{"--level without value", []string{"--level"}},
		{"--citizen without value", []string{"--citizen"}},
		{"unknown flag", []string{"--bogus", "."}},
		{"invalid level", []string{"--level", "extreme", "."}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := runCheck(nil, tt.args)
			if code != 1 {
				t.Fatalf("runCheck(%v) = %d, want 1", tt.args, code)
			}
		})
	}
}

func TestRunCity_MissingRepo(t *testing.T) {
	oldErr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() { os.Stderr = oldErr }()

	code := runCity(nil, []string{})
	if code != 3 {
		t.Fatalf("runCity with no repo = %d, want 3 (ExitInvalid)", code)
	}
}

func TestRunCity_FlagErrors(t *testing.T) {
	oldErr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() { os.Stderr = oldErr }()

	tests := []struct {
		name string
		args []string
		want int
	}{
		{"--install-at without value", []string{"--install-at"}, 3},
		{"--standalone-timeout without value", []string{"--standalone-timeout"}, 3},
		{"--standalone-timeout invalid", []string{"--standalone-timeout", "nope", "."}, 3},
		{"--standalone-timeout negative", []string{"--standalone-timeout", "-5s", "."}, 3},
		{"--citizen without value", []string{"--citizen"}, 3},
		{"unknown flag", []string{"--bogus", "."}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := runCity(nil, tt.args)
			if code != tt.want {
				t.Fatalf("runCity(%v) = %d, want %d", tt.args, code, tt.want)
			}
		})
	}
}

func TestRunHistory_FlagErrors(t *testing.T) {
	oldErr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() { os.Stderr = oldErr }()

	tests := []struct {
		name string
		args []string
	}{
		{"--repo without value", []string{"--repo"}},
		{"--citizen without value", []string{"--citizen"}},
		{"--limit without value", []string{"--limit"}},
		{"--limit zero", []string{"--limit", "0"}},
		{"--limit negative", []string{"--limit", "-5"}},
		{"--limit non-numeric", []string{"--limit", "abc"}},
		{"unknown flag", []string{"--bogus"}},
		{"--repo invalid chars", []string{"--repo", "foo/bar"}},
		{"--citizen invalid chars", []string{"--citizen", "a b c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := runHistory(tt.args)
			if code != 1 {
				t.Fatalf("runHistory(%v) = %d, want 1", tt.args, code)
			}
		})
	}
}
