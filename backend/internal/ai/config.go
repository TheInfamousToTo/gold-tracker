package ai

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultModel        = "claude-opus-5"
	defaultTimeout      = 180 * time.Second
	defaultAutoMinHours = 24.0
)

type Config struct {
	Enabled      bool
	Model        string
	Timeout      time.Duration
	AutoMinHours float64
}

// LoadConfigFromEnv reads the AI_* settings. Every value is optional so
// the application boots normally with the feature switched off.
//
// Both AI_ENABLED and a token are required: the flag alone would leave
// the CLI unable to authenticate inside a container, which would turn
// every generation attempt into a failure the owner has to diagnose.
func LoadConfigFromEnv() Config {
	enabled := os.Getenv("AI_ENABLED") == "true" && os.Getenv("CLAUDE_CODE_OAUTH_TOKEN") != ""

	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = defaultModel
	}

	timeout := defaultTimeout
	if v := os.Getenv("AI_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Second
		}
	}

	autoMinHours := defaultAutoMinHours
	if v := os.Getenv("AI_AUTO_MIN_HOURS"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n >= 0 {
			autoMinHours = n
		}
	}

	return Config{
		Enabled:      enabled,
		Model:        model,
		Timeout:      timeout,
		AutoMinHours: autoMinHours,
	}
}
