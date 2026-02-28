package gates

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunCmdImpl_Success(t *testing.T) {
	pass, output, err := runCmdImpl(context.Background(), t.TempDir(), 10, "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pass {
		t.Fatal("expected pass")
	}
	if !strings.Contains(output, "hello") {
		t.Fatalf("expected 'hello' in output, got %q", output)
	}
}

func TestRunCmdImpl_NonZeroExit(t *testing.T) {
	pass, _, err := runCmdImpl(context.Background(), t.TempDir(), 10, "false")
	if err != nil {
		t.Fatalf("non-zero exit should not be an error: %v", err)
	}
	if pass {
		t.Fatal("expected fail for non-zero exit")
	}
}

func TestRunCmdImpl_CommandNotFound(t *testing.T) {
	pass, _, err := runCmdImpl(context.Background(), t.TempDir(), 10, "nonexistent-cmd-12345")
	if err == nil {
		t.Fatal("expected error for missing command")
	}
	if pass {
		t.Fatal("expected fail")
	}
	if !strings.Contains(err.Error(), "exec nonexistent-cmd-12345") {
		t.Fatalf("expected exec error, got: %v", err)
	}
}

func TestRunCmdImpl_Timeout(t *testing.T) {
	// Use a short parent context so the test doesn't wait long.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	pass, _, err := runCmdImpl(ctx, t.TempDir(), 300, "sleep", "30")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if pass {
		t.Fatal("expected fail on timeout")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestRunCmdImpl_CapturesStderr(t *testing.T) {
	pass, output, err := runCmdImpl(context.Background(), t.TempDir(), 10, "bash", "-c", "echo stderr-msg >&2; exit 1")
	if err != nil {
		t.Fatalf("non-zero exit should not be an error: %v", err)
	}
	if pass {
		t.Fatal("expected fail")
	}
	if !strings.Contains(output, "stderr-msg") {
		t.Fatalf("expected stderr in output, got %q", output)
	}
}

func TestRunCmdImpl_UsesDir(t *testing.T) {
	dir := t.TempDir()
	pass, output, err := runCmdImpl(context.Background(), dir, 10, "pwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pass {
		t.Fatal("expected pass")
	}
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		t.Fatal("expected non-empty output from pwd")
	}
}
