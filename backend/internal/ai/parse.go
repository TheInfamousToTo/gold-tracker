package ai

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// Advice is the model's verdict for one metal.
type Advice struct {
	Action      string           `json:"action"`
	AmountBHD   float64          `json:"amount_bhd"`
	Confidence  float64          `json:"confidence"`
	Reasoning   string           `json:"reasoning"`
	HorizonDays int              `json:"horizon_days"`
	KeyFactors  []string         `json:"key_factors"`
	News        []model.NewsItem `json:"news"`
}

// Review is the model's answer to a plan the owner typed.
type Review struct {
	Verdict    string           `json:"verdict"`
	Confidence float64          `json:"confidence"`
	Reasoning  string           `json:"reasoning"`
	KeyFactors []string         `json:"key_factors"`
	News       []model.NewsItem `json:"news"`
	Suggested  *Suggestion      `json:"suggested"`
}

// Suggestion is what a review recommends doing instead, or as well.
type Suggestion struct {
	Metal     string  `json:"metal"`
	Action    string  `json:"action"`
	AmountBHD float64 `json:"amount_bhd"`
}

const (
	VerdictGood   = "GOOD_IDEA"
	VerdictAdjust = "ADJUST"
	VerdictBad    = "BAD_IDEA"
)

// maxReasoningLen is what a card can show without turning into an
// essay. The prompt asks for 450; anything longer is trimmed at a
// sentence boundary rather than rejected, since a rerun costs minutes
// and a slice of shared subscription quota.
const maxReasoningLen = 600

// absurdReasoningLen is the point at which the response is treated as
// broken rather than merely verbose.
const absurdReasoningLen = 8000

const maxNews = 5

var validActions = map[string]bool{ActionBuy: true, ActionHold: true, ActionSell: true, ActionCutLoss: true}
var validVerdicts = map[string]bool{VerdictGood: true, VerdictAdjust: true, VerdictBad: true}
var validImpacts = map[string]bool{"bullish": true, "bearish": true, "neutral": true}

// ParseAnalysis extracts the first balanced JSON object from raw and
// validates an Advice for every metal named. Each metal must appear as
// its own key: gold and silver are separate markets, and a run that
// answered for one cannot be silently applied to the other.
//
// Headless CLI invocation has no equivalent of the API's structured
// output, so the schema is enforced here. The enum checks also mean a
// prompt injection that survives into the output — from a web page the
// model read, say — still cannot produce an arbitrary action.
func ParseAnalysis(raw string, metals []string) (map[string]Advice, error) {
	fields, err := objectFields(raw)
	if err != nil {
		return nil, err
	}
	out := make(map[string]Advice, len(metals))
	for _, metal := range metals {
		body, ok := fields[metal]
		if !ok {
			return nil, fmt.Errorf("response has no verdict for %s", metal)
		}
		var a Advice
		if err := json.Unmarshal(body, &a); err != nil {
			return nil, fmt.Errorf("%s: verdict is not valid JSON: %w", metal, err)
		}
		if !validActions[a.Action] {
			return nil, fmt.Errorf("%s: action %q is not one of BUY, HOLD, SELL, CUT_LOSS", metal, a.Action)
		}
		if err := checkCommon(&a.Confidence, &a.Reasoning, &a.KeyFactors, &a.News); err != nil {
			return nil, fmt.Errorf("%s: %w", metal, err)
		}
		if a.AmountBHD < 0 {
			a.AmountBHD = 0
		}
		out[metal] = a
	}
	return out, nil
}

// ParseReview validates a plan review.
func ParseReview(raw string) (Review, error) {
	jsonStr, err := extractFirstJSONObject(raw)
	if err != nil {
		return Review{}, err
	}
	var r Review
	if err := json.Unmarshal([]byte(jsonStr), &r); err != nil {
		return Review{}, fmt.Errorf("review is not valid JSON: %w", err)
	}
	if !validVerdicts[r.Verdict] {
		return Review{}, fmt.Errorf("verdict %q is not one of GOOD_IDEA, ADJUST, BAD_IDEA", r.Verdict)
	}
	if err := checkCommon(&r.Confidence, &r.Reasoning, &r.KeyFactors, &r.News); err != nil {
		return Review{}, err
	}
	if s := r.Suggested; s != nil {
		if !model.IsKnownMetal(s.Metal) || !validActions[s.Action] {
			// A malformed suggestion does not sink an otherwise good
			// review; it is dropped and the verdict stands.
			r.Suggested = nil
		} else if s.AmountBHD < 0 {
			s.AmountBHD = 0
		}
	}
	return r, nil
}

func objectFields(raw string) (map[string]json.RawMessage, error) {
	jsonStr, err := extractFirstJSONObject(raw)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &fields); err != nil {
		return nil, fmt.Errorf("response is not valid JSON: %w", err)
	}
	return fields, nil
}

func checkCommon(confidence *float64, reasoning *string, factors *[]string, news *[]model.NewsItem) error {
	if *confidence < 0 || *confidence > 1 {
		return fmt.Errorf("confidence %v is not in [0,1]", *confidence)
	}
	if strings.TrimSpace(*reasoning) == "" {
		return fmt.Errorf("reasoning is empty")
	}
	if len(*reasoning) > absurdReasoningLen {
		return fmt.Errorf("reasoning is %d chars, over the %d char limit", len(*reasoning), absurdReasoningLen)
	}
	*reasoning = trimToLength(*reasoning, maxReasoningLen)
	*factors = trimFactors(*factors)
	*news = cleanNews(*news)
	return nil
}

// cleanNews keeps at most maxNews items, and only those whose URL is a
// real http(s) link: a citation nobody can open is not evidence.
func cleanNews(items []model.NewsItem) []model.NewsItem {
	out := make([]model.NewsItem, 0, len(items))
	for _, n := range items {
		u, err := url.Parse(strings.TrimSpace(n.URL))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			continue
		}
		n.URL = u.String()
		n.Title = trimToLength(n.Title, 160)
		n.Source = trimToLength(n.Source, 60)
		n.Date = trimToLength(n.Date, 10)
		n.Impact = strings.ToLower(strings.TrimSpace(n.Impact))
		if !validImpacts[n.Impact] {
			n.Impact = "neutral"
		}
		if n.Title == "" {
			continue
		}
		out = append(out, n)
		if len(out) == maxNews {
			break
		}
	}
	return out
}

// trimToLength shortens s to at most limit characters, preferring to
// end on the last complete sentence so the reasoning still reads as
// finished rather than cut off mid-clause.
func trimToLength(s string, limit int) string {
	s = strings.TrimSpace(s)
	if len(s) <= limit {
		return s
	}
	// Byte slicing can land mid-rune, and the reasoning routinely
	// contains em dashes; drop any partial rune the cut created.
	cut := strings.ToValidUTF8(s[:limit], "")
	if i := strings.LastIndexAny(cut, ".!?"); i > limit/2 {
		return strings.TrimSpace(cut[:i+1])
	}
	if i := strings.LastIndexByte(cut, ' '); i > limit/2 {
		return strings.TrimSpace(cut[:i]) + "…"
	}
	return strings.TrimSpace(cut) + "…"
}

// trimFactors caps the supporting points at three, each short enough to
// sit on one line.
func trimFactors(factors []string) []string {
	const (
		maxFactors   = 3
		maxFactorLen = 80
	)
	if len(factors) > maxFactors {
		factors = factors[:maxFactors]
	}
	out := make([]string, 0, len(factors))
	for _, f := range factors {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		out = append(out, trimToLength(f, maxFactorLen))
	}
	return out
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
