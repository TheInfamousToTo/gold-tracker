package ai

import (
	"strings"
	"testing"
)

// The distinctive opening of the sparse-data warning. Asserting on the
// bare word "hedge" is not enough: the prompt also tells the model not
// to hedge both ways, unconditionally.
const sparseWarning = "Fewer than 14 observations"

// gold builds the single-metal input that most of these tests use.
func gold(prices []PriceHistoryPoint, holdings ...HoldingsAggregate) PromptInput {
	return PromptInput{
		Metals: []MetalData{{Metal: "gold", Prices: prices, Holdings: holdings}},
	}
}

func TestBuildPromptIncludesSchemaInstruction(t *testing.T) {
	prompt := BuildPrompt(gold(nil))
	for _, want := range []string{`"signal"`, "BUY", "SELL", "HOLD", "confidence", "reasoning"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestBuildPromptIncludesPriceHistory(t *testing.T) {
	prompt := BuildPrompt(gold([]PriceHistoryPoint{
		{Date: "2026-08-01", PricePerGram: 45.123},
		{Date: "2026-08-02", PricePerGram: 45.500},
	}))
	if !strings.Contains(prompt, "2026-08-01") || !strings.Contains(prompt, "45.123") {
		t.Fatalf("prompt missing price history:\n%s", prompt)
	}
}

func TestBuildPromptIncludesHoldings(t *testing.T) {
	prompt := BuildPrompt(gold(nil,
		HoldingsAggregate{PurityLabel: "21K", TotalWeightGrams: 50, TotalPaid: 2000, AvgPricePerGram: 40},
	))
	if !strings.Contains(prompt, "21K") || !strings.Contains(prompt, "50.00") {
		t.Fatalf("prompt missing holdings aggregate:\n%s", prompt)
	}
}

func TestBuildPromptHedgesOnSparseData(t *testing.T) {
	prompt := BuildPrompt(gold([]PriceHistoryPoint{{Date: "2026-08-01", PricePerGram: 45}}))
	if !strings.Contains(prompt, sparseWarning) {
		t.Fatalf("prompt should carry the sparse-data warning:\n%s", prompt)
	}
}

func TestBuildPromptDoesNotHedgeOnDenseData(t *testing.T) {
	var prices []PriceHistoryPoint
	for i := 0; i < 30; i++ {
		prices = append(prices, PriceHistoryPoint{Date: "2026-08-01", PricePerGram: 45})
	}
	prompt := BuildPrompt(gold(prices))
	if strings.Contains(prompt, sparseWarning) {
		t.Fatalf("prompt should not carry the sparse-data warning with 30 observations:\n%s", prompt)
	}
}

func TestBuildPromptFramesDataAsDataNotInstructions(t *testing.T) {
	prompt := BuildPrompt(gold(nil))
	if !strings.Contains(prompt, "not instructions") {
		t.Fatalf("prompt should frame the payload as data, not instructions:\n%s", prompt)
	}
}

// PromptInput carries no field for item_name, vendor, or notes, so
// owner-typed free text structurally cannot reach the model. This test
// documents that guarantee.
func TestBuildPromptNeverReferencesFreeTextFields(t *testing.T) {
	prompt := BuildPrompt(gold(
		[]PriceHistoryPoint{{Date: "2026-08-01", PricePerGram: 45}},
		HoldingsAggregate{PurityLabel: "21K", TotalWeightGrams: 10, TotalPaid: 400, AvgPricePerGram: 40},
	))
	for _, forbidden := range []string{"vendor", "notes", "item_name"} {
		if strings.Contains(strings.ToLower(prompt), forbidden) {
			t.Errorf("prompt should never reference %q:\n%s", forbidden, prompt)
		}
	}
}

func TestBuildPromptIncludesPortfolioTotals(t *testing.T) {
	in := gold(nil)
	in.TotalPaid = 1000
	in.TotalValue = 1200
	in.TotalGainLossPct = 20

	prompt := BuildPrompt(in)
	if !strings.Contains(prompt, "1000.000") || !strings.Contains(prompt, "1200.000") {
		t.Fatalf("prompt missing portfolio totals:\n%s", prompt)
	}
}

func TestBuildPromptIncludesDerivedStatistics(t *testing.T) {
	var prices []PriceHistoryPoint
	for i := 0; i < 40; i++ {
		prices = append(prices, PriceHistoryPoint{Date: "2026-08-01", PricePerGram: 45 + float64(i)*0.1})
	}
	prompt := BuildPrompt(gold(prices))
	for _, want := range []string{"Derived statistics", "latest:", "mean of last", "range of full series"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestBuildPromptFlagsCarriedForwardPrints(t *testing.T) {
	prompt := BuildPrompt(gold([]PriceHistoryPoint{
		{Date: "2026-08-01", PricePerGram: 45},
		{Date: "2026-08-02", PricePerGram: 45},
		{Date: "2026-08-03", PricePerGram: 46},
	}))
	if !strings.Contains(prompt, "carry-forwards") {
		t.Fatalf("prompt should flag repeated prints:\n%s", prompt)
	}
}

func TestBuildPromptCapsReasoningLength(t *testing.T) {
	prompt := BuildPrompt(gold(nil))
	if !strings.Contains(prompt, "320 characters") {
		t.Fatalf("prompt should ask for a short reasoning:\n%s", prompt)
	}
}

// A gold-only owner must keep seeing the flat single-metal schema. If
// the prompt asked for {"gold": {...}} the response would parse, but
// every existing install would change shape for no reason.
func TestBuildPromptKeepsFlatSchemaForOneMetal(t *testing.T) {
	prompt := BuildPrompt(gold([]PriceHistoryPoint{{Date: "2026-08-01", PricePerGram: 45}}))
	if !strings.Contains(prompt, verdictShape) {
		t.Fatalf("single-metal prompt should ask for the flat verdict:\n%s", prompt)
	}
	if strings.Contains(prompt, `"silver"`) {
		t.Fatalf("single-metal prompt should not mention silver:\n%s", prompt)
	}
}

func TestBuildPromptAsksForAVerdictPerMetal(t *testing.T) {
	prompt := BuildPrompt(PromptInput{
		Metals: []MetalData{
			{Metal: "gold", Prices: []PriceHistoryPoint{{Date: "2026-08-01", PricePerGram: 45}}},
			{Metal: "silver", Prices: []PriceHistoryPoint{{Date: "2026-08-01", PricePerGram: 0.41}}},
		},
	})
	for _, want := range []string{`"gold":`, `"silver":`, "== GOLD ==", "== SILVER =="} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

// Silver is quoted at 999, not 24K. Labelling its series 24K would
// invite the model to reason about it as if it were gold.
func TestBuildPromptLabelsSilverSeriesAs999(t *testing.T) {
	prompt := BuildPrompt(PromptInput{
		Metals: []MetalData{
			{Metal: "silver", Prices: []PriceHistoryPoint{{Date: "2026-08-01", PricePerGram: 0.41}}},
		},
	})
	if !strings.Contains(prompt, "999 BHD per gram") {
		t.Fatalf("silver series should be labelled 999:\n%s", prompt)
	}
}

func TestBuildPromptTellsTheModelNotToCopyVerdictsAcross(t *testing.T) {
	prompt := BuildPrompt(PromptInput{
		Metals: []MetalData{{Metal: "gold"}, {Metal: "silver"}},
	})
	if !strings.Contains(prompt, "do not copy one verdict across") {
		t.Fatalf("prompt should warn against copying a verdict between metals:\n%s", prompt)
	}
}

func TestBuildPromptStatesWhenAMetalIsNotHeld(t *testing.T) {
	prompt := BuildPrompt(PromptInput{
		Metals: []MetalData{
			{Metal: "silver", Prices: []PriceHistoryPoint{{Date: "2026-08-01", PricePerGram: 0.41}}},
		},
	})
	if !strings.Contains(prompt, "none held") {
		t.Fatalf("prompt should say when a metal has no holdings:\n%s", prompt)
	}
}
