//go:build aiprobe

// This file answers one question: given clearly directional data, does
// the model actually return BUY and SELL, or does it hedge to HOLD
// whatever it is shown?
//
// Nothing in the code path forces a HOLD — ParseVerdicts rejects any
// signal outside the enum, and a failed run saves nothing at all — so
// a HOLD in the log came from the model. What the code cannot tell us
// is whether the prompt's caution has made HOLD the only reachable
// answer in practice.
//
// It runs the real CLI against the real prompt, so it spends
// subscription quota and is kept behind a build tag:
//
//	go test -tags aiprobe ./internal/ai/ -run TestProbe -v -timeout 20m
//
// Read the output rather than the pass/fail: the test only fails if
// every scenario returns the same verdict, which is the specific
// failure worth catching automatically.
package ai

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// series builds a daily price series from a starting price and a
// per-day drift, with a small deterministic wobble so it does not look
// synthetic enough to be dismissed as one.
func series(start, driftPct float64, days int) []PriceHistoryPoint {
	points := make([]PriceHistoryPoint, 0, days)
	price := start
	day := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < days; i++ {
		wobble := math.Sin(float64(i)*1.7) * start * 0.002
		points = append(points, PriceHistoryPoint{
			Date:         day.AddDate(0, 0, i).Format("2006-01-02"),
			PricePerGram: price + wobble,
		})
		price *= 1 + driftPct/100
	}
	return points
}

// goldInput runs the series through the same quant and rules the
// service uses, holding 50 g of 24K bought at 28 BHD/g, with a 100 BHD
// monthly budget and news off so the probe judges the data alone.
func goldInput(prices []PriceHistoryPoint) AdvisorInput {
	settings := model.DefaultAdvisorSettings()
	settings.MonthlyBudgetBHD = 100
	settings.NewsEnabled = false
	ind := computeIndicators(prices)
	value := 50 * ind.Latest
	pos := Position{Held: true, FineGrams: 50, Paid: 1400, Value: value, NetValue: value * 0.99,
		NetPLPct: (value*0.99/1400 - 1) * 100, Share: 1}
	states := []MetalState{{Metal: "gold", Prices: prices, Indicators: ind, Position: pos}}
	b := Budget{Monthly: 100, Remaining: 100}
	computeBaselines(states, settings, b, RatioStats{})
	return AdvisorInput{Today: prices[len(prices)-1].Date, Metals: states, Budget: b, Settings: settings}
}

func TestProbeSignalSpread(t *testing.T) {
	scenarios := []struct {
		name   string
		input  AdvisorInput
		expect string // what a competent advisor should say; not asserted
	}{
		{"sustained rally, price far above entry", goldInput(series(28, 0.3, 400)), "small BUY, or SELL (trim)"},
		{"sustained slide below entry, trend broken", goldInput(series(45, -0.15, 400)), "CUT_LOSS or BUY"},
		{"flat drift", goldInput(series(32, 0.005, 400)), "BUY at 1x"},
		{"sparse history", goldInput(series(32, 0.5, 20)), "low confidence"},
		{"sharp recent crash after a long rise", goldInput(append(series(28, 0.2, 380), series(60, -2.0, 20)...)), "BUY extra"},
	}

	runner := &CLIRunner{Timeout: 5 * time.Minute}
	seen := map[string]int{}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			result, err := runner.Run(context.Background(), BuildAnalysisPrompt(sc.input), "claude-opus-5-5", nil)
			if err != nil {
				t.Fatalf("runner: %v", err)
			}
			if result.IsError {
				t.Fatalf("CLI reported an error: %s", result.Result)
			}
			advice, err := ParseAnalysis(result.Result, []string{"gold"})
			if err != nil {
				t.Fatalf("parse: %v\nraw: %s", err, result.Result)
			}
			a := advice["gold"]
			seen[a.Action]++
			b := sc.input.Metals[0].Baseline
			fmt.Printf("\n--- %s\n    expected roughly: %s\n    rules: %s %.2f\n    got: %s %.2f (confidence %.2f)\n    %s\n",
				sc.name, sc.expect, b.Action, b.AmountBHD, a.Action, a.AmountBHD, a.Confidence, a.Reasoning)
		})
	}

	fmt.Printf("\n=== verdict spread across %d scenarios: %v\n", len(scenarios), seen)

	// The narrow question this probe exists to answer. One verdict for
	// every scenario means the prompt, not the market, is deciding.
	if len(seen) == 1 {
		for action := range seen {
			t.Errorf("every scenario returned %s — the model is not discriminating between them", action)
		}
	}
}
