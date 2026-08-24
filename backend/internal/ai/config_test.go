package ai

import (
	"testing"
	"time"
)

func clearAIEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AI_ENABLED", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("AI_MODEL", "")
	t.Setenv("AI_TIMEOUT_SECONDS", "")
	t.Setenv("AI_AUTO_MIN_HOURS", "")
}

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	clearAIEnv(t)

	cfg := LoadConfigFromEnv()
	if cfg.Enabled {
		t.Errorf("Enabled = true, want false when AI_ENABLED and the token are unset")
	}
	if cfg.Model != "claude-opus-5" {
		t.Errorf("Model = %q, want default claude-opus-5", cfg.Model)
	}
	if cfg.Timeout != 180*time.Second {
		t.Errorf("Timeout = %v, want default 180s", cfg.Timeout)
	}
	if cfg.AutoMinHours != 24 {
		t.Errorf("AutoMinHours = %v, want default 24", cfg.AutoMinHours)
	}
}

func TestLoadConfigFromEnvExplicit(t *testing.T) {
	clearAIEnv(t)
	t.Setenv("AI_ENABLED", "true")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "fake-token")
	t.Setenv("AI_MODEL", "claude-sonnet-5")
	t.Setenv("AI_TIMEOUT_SECONDS", "60")
	t.Setenv("AI_AUTO_MIN_HOURS", "12")

	cfg := LoadConfigFromEnv()
	if !cfg.Enabled {
		t.Errorf("Enabled = false, want true")
	}
	if cfg.Model != "claude-sonnet-5" {
		t.Errorf("Model = %q, want claude-sonnet-5", cfg.Model)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want 60s", cfg.Timeout)
	}
	if cfg.AutoMinHours != 12 {
		t.Errorf("AutoMinHours = %v, want 12", cfg.AutoMinHours)
	}
}

// AI_ENABLED is the only gate. The CLI resolves credentials itself,
// either from the token or from a logged-in session on the host, so
// requiring the token here would wrongly disable the feature on a
// developer machine that is already authenticated.
func TestLoadConfigEnabledGatesOnFlagAlone(t *testing.T) {
	clearAIEnv(t)
	t.Setenv("AI_ENABLED", "true")

	if !LoadConfigFromEnv().Enabled {
		t.Error("Enabled = false with AI_ENABLED=true and no token, want true")
	}

	clearAIEnv(t)
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "a-token")

	if LoadConfigFromEnv().Enabled {
		t.Error("Enabled = true with AI_ENABLED unset, want false")
	}
}

func TestLoadConfigIgnoresUnparseableNumbers(t *testing.T) {
	clearAIEnv(t)
	t.Setenv("AI_TIMEOUT_SECONDS", "not-a-number")
	t.Setenv("AI_AUTO_MIN_HOURS", "also-not-a-number")

	cfg := LoadConfigFromEnv()
	if cfg.Timeout != 180*time.Second {
		t.Errorf("Timeout = %v, want the 180s default to survive a bad value", cfg.Timeout)
	}
	if cfg.AutoMinHours != 24 {
		t.Errorf("AutoMinHours = %v, want the 24 default to survive a bad value", cfg.AutoMinHours)
	}
}
