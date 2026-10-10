package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RunResult is the outcome of one model invocation. IsError reports a
// failure the CLI itself described (bad model, auth problem); a
// transport failure — the binary missing, a timeout, unparseable
// output — comes back as a Go error instead.
type RunResult struct {
	IsError bool
	Result  string
	Subtype string
}

// Runner invokes a model and returns its final text result. The
// service depends on this interface so its tests never spawn a real
// process.
type Runner interface {
	Run(ctx context.Context, prompt string, model string, tools []string) (RunResult, error)
}

// NewsTools are the only tools a run may be given: the model can read
// the web and nothing else — no shell, no filesystem.
var NewsTools = []string{"WebSearch", "WebFetch"}

// CLIRunner spawns the Claude Code CLI in headless mode. Authentication
// comes from CLAUDE_CODE_OAUTH_TOKEN in the inherited process
// environment, so calls run on the owner's subscription rather than
// against the metered Messages API.
type CLIRunner struct {
	Timeout time.Duration

	// binary overrides the executable name in tests.
	binary string
}

// cliEnvelope is the subset of the CLI's --output-format json payload
// that matters here. The CLI emits this same shape on both success
// (exit 0) and its own reported failures (exit 1).
type cliEnvelope struct {
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
	Subtype string `json:"subtype"`
}

// cliArgs builds the headless invocation. --tools sets which built-in
// tools exist at all, --allowedTools pre-approves them (there is nobody
// to answer a permission prompt), and --strict-mcp-config keeps any MCP
// server a config file might name out of the run.
func cliArgs(prompt, model string, tools []string) []string {
	list := strings.Join(tools, ",")
	return []string{
		"-p", prompt,
		"--output-format", "json",
		"--model", model,
		"--tools", list,
		"--allowedTools", list,
		"--strict-mcp-config",
	}
}

func (c *CLIRunner) Run(ctx context.Context, prompt string, model string, tools []string) (RunResult, error) {
	binary := c.binary
	if binary == "" {
		binary = "claude"
	}

	runCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, binary, cliArgs(prompt, model, tools)...)
	cmd.Stdin = nil

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	if runCtx.Err() == context.DeadlineExceeded {
		return RunResult{}, fmt.Errorf("claude CLI timed out after %s", c.Timeout)
	}
	if ctx.Err() != nil {
		return RunResult{}, fmt.Errorf("claude CLI cancelled: %w", ctx.Err())
	}

	// Parse whatever the CLI wrote before treating a non-zero exit as
	// fatal: it reports its own errors through the same JSON envelope.
	if stdout.Len() == 0 {
		if runErr != nil {
			return RunResult{}, fmt.Errorf("claude CLI failed to run: %w (stderr: %s)", runErr, stderr.String())
		}
		return RunResult{}, fmt.Errorf("claude CLI produced no output (stderr: %s)", stderr.String())
	}

	var env cliEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		return RunResult{}, fmt.Errorf("could not parse claude CLI output as JSON: %w", err)
	}

	return RunResult{IsError: env.IsError, Result: env.Result, Subtype: env.Subtype}, nil
}
