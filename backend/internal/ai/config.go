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
// AI_ENABLED alone is the gate. The CLI resolves its own credentials —
// from CLAUDE_CODE_OAUTH_TOKEN, or from an existing logged-in session
// on the host — so the backend does not check for a token. In a
// container there is no session, which is why the token is required
// there; setting it to a placeholder is worse than leaving it unset,
// because it overrides working session auth and produces a 401.
func LoadConfigFromEnv() Config {
	enabled := os.Getenv("AI_ENABLED") == "true"

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
