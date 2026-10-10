package ai

import (
	"strings"
	"testing"
)

func withBaseline(s MetalState, b Baseline) MetalState {
	s.Baseline = b
	return s
}

func TestClampCapsABuyAtOneAndAHalfTimesTheBaseline(t *testing.T) {
	s := withBaseline(state("gold", flat(0), Position{}), Baseline{Action: ActionBuy, AmountBHD: 100, Share: 1})
	c := clampAdvice(Advice{Action: ActionBuy, AmountBHD: 500}, s, settingsWith(nil), budget(100, 0, 1000))
	if c.AmountBHD != 150 {
		t.Fatalf("amount = %v, want 150", c.AmountBHD)
	}
	near(t, "grams", c.AmountGrams, round3v(150/(50*1.01)), 1e-9)
	if !c.Deviates {
		t.Error("a 150 BHD buy against a 100 BHD baseline should be marked as departing")
	}
}

func TestClampNeverSpendsMoreThanBudgetPlusReserve(t *testing.T) {
	s := withBaseline(state("gold", flat(0), Position{}), Baseline{Action: ActionBuy, AmountBHD: 100, Share: 1})
	c := clampAdvice(Advice{Action: ActionBuy, AmountBHD: 140}, s, settingsWith(nil), budget(100, 0, 20))
	if c.AmountBHD != 120 {
		t.Fatalf("amount = %v, want 120 (100 left + 20 reserve)", c.AmountBHD)
	}
}

func TestClampLimitsABuyTheRulesDidNotCallFor(t *testing.T) {
	s := withBaseline(state("gold", flat(0), Position{}), Baseline{Action: ActionHold, Share: 0.6})
	c := clampAdvice(Advice{Action: ActionBuy, AmountBHD: 90}, s, settingsWith(nil), budget(100, 0, 0))
	if c.AmountBHD != 60 {
		t.Fatalf("amount = %v, want the normal 60 BHD share", c.AmountBHD)
	}
}

func TestClampTurnsASellOfNothingIntoHold(t *testing.T) {
	s := withBaseline(state("gold", flat(0), Position{}), Baseline{Action: ActionBuy, AmountBHD: 100, Share: 1})
	for _, action := range []string{ActionSell, ActionCutLoss} {
		c := clampAdvice(Advice{Action: action, AmountBHD: 100, Confidence: 0.9}, s, settingsWith(nil), budget(100, 0, 0))
		if c.Action != ActionHold || c.AmountBHD != 0 {
			t.Errorf("%s with nothing held: got %s %v, want HOLD 0", action, c.Action, c.AmountBHD)
		}
	}
}

func TestClampCapsAContrarySellsConfidenceAndSize(t *testing.T) {
	pos := Position{Held: true, FineGrams: 20, NetValue: 990, Value: 1000, Paid: 900}
	s := withBaseline(state("gold", flat(0), pos), Baseline{Action: ActionBuy, AmountBHD: 100, Share: 1})
	c := clampAdvice(Advice{Action: ActionCutLoss, AmountBHD: 5000, Confidence: 0.95}, s, settingsWith(nil), budget(100, 0, 0))
	if c.Action != ActionCutLoss || c.AmountBHD != 990 {
		t.Fatalf("got %s %v, want CUT_LOSS capped at the 990 BHD held", c.Action, c.AmountBHD)
	}
	if c.Confidence != contraryConfidenceCap {
		t.Errorf("confidence = %v, want capped at %v", c.Confidence, contraryConfidenceCap)
	}
	if c.AmountGrams > 20 {
		t.Errorf("grams = %v, more than the 20 g held", c.AmountGrams)
	}
}

func TestClampKeepsAgreedSellConfidence(t *testing.T) {
	pos := Position{Held: true, FineGrams: 20, NetValue: 990, Value: 1000}
	s := withBaseline(state("gold", flat(0), pos), Baseline{Action: ActionSell, AmountBHD: 200})
	c := clampAdvice(Advice{Action: ActionSell, Confidence: 0.8}, s, settingsWith(nil), budget(0, 0, 0))
	if c.AmountBHD != 200 || c.Confidence != 0.8 {
		t.Fatalf("got %v BHD at %v, want the baseline's 200 at 0.8", c.AmountBHD, c.Confidence)
	}
}

func TestCapTotalBuysScalesBothMetalsTogether(t *testing.T) {
	advice := map[string]*Clamped{
		"gold":   {Advice: Advice{Action: ActionBuy, AmountBHD: 150}},
		"silver": {Advice: Advice{Action: ActionBuy, AmountBHD: 50}},
	}
	capTotalBuys(advice, budget(100, 0, 0), map[string]float64{"gold": 50, "silver": 0.7}, settingsWith(nil))
	if advice["gold"].AmountBHD != 75 || advice["silver"].AmountBHD != 25 {
		t.Fatalf("got gold %v silver %v, want 75 / 25", advice["gold"].AmountBHD, advice["silver"].AmountBHD)
	}
}

func TestParseAnalysisReadsEachMetalAndCleansNews(t *testing.T) {
	raw := "Here you go:\n```json\n" + `{"gold": {"action": "BUY", "amount_bhd": 40, "confidence": 0.7,
	  "reasoning": "Buy {steady}.", "horizon_days": 180, "key_factors": ["a","b","c","d"],
	  "news": [{"title": "Fed holds", "source": "Reuters", "date": "2026-10-08", "url": "https://reuters.com/x", "impact": "Bullish"},
	           {"title": "No link", "url": "javascript:alert(1)"},
	           {"title": "Relative", "url": "/news/1"}]},
	 "silver": {"action": "CUT_LOSS", "amount_bhd": 0, "confidence": 0.4, "reasoning": "Cut.", "key_factors": [], "news": []}}` + "\n```"
	got, err := ParseAnalysis(raw, []string{"gold", "silver"})
	if err != nil {
		t.Fatal(err)
	}
	g := got["gold"]
	if g.Action != ActionBuy || g.AmountBHD != 40 || len(g.KeyFactors) != 3 {
		t.Errorf("gold = %+v", g)
	}
	if len(g.News) != 1 || g.News[0].Impact != "bullish" {
		t.Errorf("news = %+v, want only the https item, impact lower-cased", g.News)
	}
	if got["silver"].Action != ActionCutLoss {
		t.Errorf("silver action = %q", got["silver"].Action)
	}
}

func TestParseAnalysisRejectsBadOutput(t *testing.T) {
	cases := map[string]string{
		"missing metal":  `{"gold": {"action": "BUY", "confidence": 0.5, "reasoning": "x"}}`,
		"unknown action": `{"gold": {"action": "YOLO", "confidence": 0.5, "reasoning": "x"}, "silver": {"action": "HOLD", "confidence": 0.5, "reasoning": "x"}}`,
		"bad confidence": `{"gold": {"action": "BUY", "confidence": 2, "reasoning": "x"}, "silver": {"action": "HOLD", "confidence": 0.5, "reasoning": "x"}}`,
		"no reasoning":   `{"gold": {"action": "BUY", "confidence": 0.5, "reasoning": " "}, "silver": {"action": "HOLD", "confidence": 0.5, "reasoning": "x"}}`,
		"no json":        `I could not decide.`,
		"absurd":         `{"gold": {"action": "BUY", "confidence": 0.5, "reasoning": "` + strings.Repeat("x", 9000) + `"}, "silver": {"action": "HOLD", "confidence": 0.5, "reasoning": "x"}}`,
	}
	for name, raw := range cases {
		if _, err := ParseAnalysis(raw, []string{"gold", "silver"}); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestParseReviewDropsAMalformedSuggestionButKeepsTheVerdict(t *testing.T) {
	r, err := ParseReview(`{"verdict": "ADJUST", "confidence": 0.6, "reasoning": "Smaller.", "key_factors": [], "news": [],
		"suggested": {"metal": "platinum", "action": "BUY", "amount_bhd": 10}}`)
	if err != nil {
		t.Fatal(err)
	}
	if r.Verdict != VerdictAdjust || r.Suggested != nil {
		t.Fatalf("got %+v, want ADJUST with the bad suggestion dropped", r)
	}
	if _, err := ParseReview(`{"verdict": "MAYBE", "confidence": 0.5, "reasoning": "x"}`); err == nil {
		t.Error("unknown verdict accepted")
	}
}
