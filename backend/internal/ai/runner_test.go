package ai

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func requireCLI(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not on PATH, skipping")
	}
}

func TestCLIRunnerSuccess(t *testing.T) {
	requireCLI(t)
	r := &CLIRunner{Timeout: 90 * time.Second}
	result, err := r.Run(context.Background(),
		`Reply with only this exact JSON, nothing else: {"ok":true}`,
		"claude-opus-5")
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected IsError=false, got true, result=%q", result.Result)
	}
	if !strings.Contains(result.Result, `"ok"`) {
		t.Errorf("Result = %q, expected it to contain the requested JSON", result.Result)
	}
}

func TestCLIRunnerInvalidModelReportsIsError(t *testing.T) {
	requireCLI(t)
	r := &CLIRunner{Timeout: 60 * time.Second}
	result, err := r.Run(context.Background(), "hi", "not-a-real-model")
	// The CLI still writes a parseable JSON envelope on this failure
	// mode, so it is a reported error, not a transport failure.
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected IsError=true for an invalid model")
	}
	if result.Result == "" {
		t.Errorf("expected a non-empty human-readable error in Result")
	}
}

func TestCLIRunnerTimeout(t *testing.T) {
	requireCLI(t)
	r := &CLIRunner{Timeout: 1 * time.Millisecond}
	_, err := r.Run(context.Background(), "hi", "claude-opus-5")
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, expected it to mention the timeout", err.Error())
	}
}

func TestCLIRunnerHonorsCallerCancellation(t *testing.T) {
	requireCLI(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := &CLIRunner{Timeout: 90 * time.Second}
	if _, err := r.Run(ctx, "hi", "claude-opus-5"); err == nil {
		t.Fatal("expected an error when the caller's context is already cancelled")
	}
}

func TestCLIRunnerMissingBinaryIsTransportError(t *testing.T) {
	r := &CLIRunner{Timeout: 5 * time.Second, binary: "definitely-not-a-real-binary-xyz"}
	_, err := r.Run(context.Background(), "hi", "claude-opus-5")
	if err == nil {
		t.Fatal("expected an error when the CLI binary is missing")
	}
}
