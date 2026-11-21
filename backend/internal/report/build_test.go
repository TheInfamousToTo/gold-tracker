package report

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

func f(v float64) *float64 { return &v }

func near(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil || math.Abs(*got-want) > 0.001 {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func TestBuild(t *testing.T) {
	in := Inputs{
		Month: "2026-10",
		Now:   time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		Gold: []model.GoldPrice{
			{PriceDate: "2026-11-05", PricePerGram24k: 52}, // today (unordered on purpose)
			{PriceDate: "2026-09-30", PricePerGram24k: 40}, // October opens here
			{PriceDate: "2026-10-15", PricePerGram24k: 44},
			{PriceDate: "2026-10-31", PricePerGram24k: 48}, // October close
		},
		Items: []model.PortfolioItem{
			// Held all month: 10 g 24K bought at 30/g.
			{ID: 1, ItemName: "Bar", MetalType: "gold", PurityKarat: f(24), WeightGrams: 10,
				PricePaidTotal: 300, PricePerGramPaid: 30, PurchaseDate: "2026-01-01", CurrentPricePerGram: f(52)},
			// Bought mid-October: 12 g 18K at 33/g.
			{ID: 2, ItemName: "Ring", MetalType: "gold", PurityKarat: f(18), WeightGrams: 12,
				PricePaidTotal: 396, PricePerGramPaid: 33, PurchaseDate: "2026-10-15", CurrentPricePerGram: f(39)},
			// Bought after the month: no month gain.
			{ID: 3, ItemName: "Coin", MetalType: "gold", PurityKarat: f(24), WeightGrams: 1,
				PricePaidTotal: 51, PricePerGramPaid: 51, PurchaseDate: "2026-11-03", CurrentPricePerGram: f(52)},
		},
		Settings: model.AdvisorSettings{MonthlyBudgetBHD: 500},
	}
	d, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}

	bar, ring, coin := d.Items[0], d.Items[1], d.Items[2]
	near(t, "bar now/g", bar.NowPerGram, 52)
	near(t, "bar gain", bar.Gain, 220)         // 520 - 300
	near(t, "bar month", bar.MonthGain, 80)    // 10 * (48 - 40)
	near(t, "ring now/g", ring.NowPerGram, 39) // 52 * 18/24
	near(t, "ring month", ring.MonthGain, 36)  // 12 * (48*0.75 - 33)
	if coin.MonthGain != nil {
		t.Fatalf("coin bought after month has month gain %v", *coin.MonthGain)
	}
	near(t, "total month", &d.Totals.MonthGain, 116)
	if len(d.Purchases) != 1 || d.SpentBHD != 396 {
		t.Fatalf("purchases = %d, spent %v", len(d.Purchases), d.SpentBHD)
	}
	if len(d.Metals) != 1 || d.Metals[0].Open != 40 || d.Metals[0].Close != 48 || d.Metals[0].ChangePct != 20 {
		t.Fatalf("metal month = %+v", d.Metals)
	}

	msgs := Messages(model.MonthlyReport{Month: "2026-10", Data: d})
	for _, m := range msgs {
		if len(m) > 2000 {
			t.Fatalf("discord message too long: %d", len(m))
		}
	}
	if !strings.Contains(msgs[1], "Ring 18K") {
		t.Fatalf("table missing ring row:\n%s", msgs[1])
	}
}

func TestSplit(t *testing.T) {
	text := strings.Repeat("line of text\n", 500) + strings.Repeat("x", 4000)
	parts := split(text, 1900)
	if strings.Join(parts, "") != text {
		t.Fatal("split lost text")
	}
	for _, p := range parts {
		if len(p) > 1900 {
			t.Fatalf("part too long: %d", len(p))
		}
	}
}

func TestPreviousMonth(t *testing.T) {
	if got := PreviousMonth(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); got != "2025-12" {
		t.Fatal(got)
	}
}

func TestPromptHidesNames(t *testing.T) {
	d := model.ReportData{Month: "2026-10", Items: []model.ReportItem{{ID: 7, Name: "Grandma ring", Purity: "21K", Metal: "gold"}}}
	p := BuildPrompt(d, model.AdvisorSettings{}, nil, false)
	if strings.Contains(p, "Grandma") || !strings.Contains(p, "Item 7 (21K gold)") {
		t.Fatal("prompt leaks item name or lacks label")
	}
}
