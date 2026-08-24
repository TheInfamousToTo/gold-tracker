package ai

import (
	"strings"
	"testing"
)

func TestBuildPromptIncludesSchemaInstruction(t *testing.T) {
	prompt := BuildPrompt(PromptInput{})
	for _, want := range []string{`"signal"`, "BUY", "SELL", "HOLD", "confidence", "reasoning"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestBuildPromptIncludesPriceHistory(t *testing.T) {
	prompt := BuildPrompt(PromptInput{
		Prices: []PriceHistoryPoint{
			{Date: "2026-08-01", PricePerGram24k: 45.123},
			{Date: "2026-08-02", PricePerGram24k: 45.500},
		},
	})
	if !strings.Contains(prompt, "2026-08-01") || !strings.Contains(prompt, "45.123") {
		t.Fatalf("prompt missing price history:\n%s", prompt)
	}
}

func TestBuildPromptIncludesHoldings(t *testing.T) {
	prompt := BuildPrompt(PromptInput{
		Holdings: []HoldingsAggregate{
			{Karat: 21, TotalWeightGrams: 50, TotalPaid: 2000, AvgPricePerGram: 40},
		},
	})
	if !strings.Contains(prompt, "21K") || !strings.Contains(prompt, "50.00") {
		t.Fatalf("prompt missing holdings aggregate:\n%s", prompt)
	}
}

func TestBuildPromptHedgesOnSparseData(t *testing.T) {
	prompt := BuildPrompt(PromptInput{
		Prices: []PriceHistoryPoint{{Date: "2026-08-01", PricePerGram24k: 45}},
	})
	if !strings.Contains(strings.ToLower(prompt), "hedge") {
		t.Fatalf("prompt should tell the model to hedge on sparse data:\n%s", prompt)
	}
}

func TestBuildPromptDoesNotHedgeOnDenseData(t *testing.T) {
	var prices []PriceHistoryPoint
	for i := 0; i < 30; i++ {
		prices = append(prices, PriceHistoryPoint{Date: "2026-08-01", PricePerGram24k: 45})
	}
	prompt := BuildPrompt(PromptInput{Prices: prices})
	if strings.Contains(strings.ToLower(prompt), "hedge") {
		t.Fatalf("prompt should not carry the sparse-data warning with 30 observations:\n%s", prompt)
	}
}

func TestBuildPromptFramesDataAsDataNotInstructions(t *testing.T) {
	prompt := BuildPrompt(PromptInput{})
	if !strings.Contains(prompt, "not instructions") {
		t.Fatalf("prompt should frame the payload as data, not instructions:\n%s", prompt)
	}
}

// PromptInput carries no field for item_name, vendor, or notes, so
// owner-typed free text structurally cannot reach the model. This test
// documents that guarantee.
func TestBuildPromptNeverReferencesFreeTextFields(t *testing.T) {
	prompt := BuildPrompt(PromptInput{
		Prices:   []PriceHistoryPoint{{Date: "2026-08-01", PricePerGram24k: 45}},
		Holdings: []HoldingsAggregate{{Karat: 21, TotalWeightGrams: 10, TotalPaid: 400, AvgPricePerGram: 40}},
	})
	for _, forbidden := range []string{"vendor", "notes", "item_name"} {
		if strings.Contains(strings.ToLower(prompt), forbidden) {
			t.Errorf("prompt should never reference %q:\n%s", forbidden, prompt)
		}
	}
}

func TestBuildPromptIncludesPortfolioTotals(t *testing.T) {
	prompt := BuildPrompt(PromptInput{
		TotalPaid:        1000,
		TotalValue:       1200,
		TotalGainLossPct: 20,
	})
	if !strings.Contains(prompt, "1000.000") || !strings.Contains(prompt, "1200.000") {
		t.Fatalf("prompt missing portfolio totals:\n%s", prompt)
	}
}
