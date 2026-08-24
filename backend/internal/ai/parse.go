package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Verdict is the recommendation the model is asked to produce.
type Verdict struct {
	Signal      string   `json:"signal"`
	Confidence  float64  `json:"confidence"`
	Reasoning   string   `json:"reasoning"`
	HorizonDays int      `json:"horizon_days"`
	KeyFactors  []string `json:"key_factors"`
}

const maxReasoningLen = 2000

var validSignals = map[string]bool{"BUY": true, "SELL": true, "HOLD": true}

// ParseVerdict extracts the first balanced JSON object from raw and
// validates it against the verdict schema.
//
// Headless CLI invocation has no equivalent of the API's
// output_config.format, so the schema is enforced here instead. The
// enum check also means a prompt injection that survives into the
// model's output still cannot produce an arbitrary signal type.
func ParseVerdict(raw string) (Verdict, error) {
	jsonStr, err := extractFirstJSONObject(raw)
	if err != nil {
		return Verdict{}, err
	}

	var v Verdict
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		return Verdict{}, fmt.Errorf("verdict is not valid JSON: %w", err)
	}

	if !validSignals[v.Signal] {
		return Verdict{}, fmt.Errorf("signal %q is not one of BUY, SELL, HOLD", v.Signal)
	}
	if v.Confidence < 0 || v.Confidence > 1 {
		return Verdict{}, fmt.Errorf("confidence %v is not in [0,1]", v.Confidence)
	}
	if strings.TrimSpace(v.Reasoning) == "" {
		return Verdict{}, fmt.Errorf("reasoning is empty")
	}
	if len(v.Reasoning) > maxReasoningLen {
		return Verdict{}, fmt.Errorf("reasoning is %d chars, over the %d char limit", len(v.Reasoning), maxReasoningLen)
	}

	return v, nil
}

// extractFirstJSONObject finds the first balanced {...} substring in s,
// respecting string literals so braces inside the reasoning text don't
// throw off the depth count.
func extractFirstJSONObject(s string) (string, error) {
	start := strings.IndexByte(s, '{')
	if start == -1 {
		return "", fmt.Errorf("no JSON object found in output")
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1], nil
			}
		}
	}
	return "", fmt.Errorf("no balanced JSON object found in output")
}
