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

func goldInput(prices []PriceHistoryPoint) PromptInput {
	return PromptInput{
		Metals: []MetalData{{
			Metal:  "gold",
			Prices: prices,
			Holdings: []HoldingsAggregate{
				{PurityLabel: "21K", TotalWeightGrams: 50, TotalPaid: 1400, AvgPricePerGram: 28},
			},
		}},
		TotalPaid:        1400,
		TotalValue:       1500,
		TotalGainLossPct: 7.14,
	}
}

func TestProbeSignalSpread(t *testing.T) {
	scenarios := []struct {
		name   string
		input  PromptInput
		expect string // what a competent analyst should say; not asserted
	}{
		{
			name:   "sustained rally, price far above entry",
			input:  goldInput(series(28, 0.9, 90)),
			expect: "SELL or HOLD",
		},
		{
			name:   "sustained slide, price far below entry",
			input:  goldInput(series(45, -0.8, 90)),
			expect: "BUY",
		},
		{
			name:   "flat drift",
			input:  goldInput(series(32, 0.01, 90)),
			expect: "HOLD",
		},
		{
			name:   "sparse history",
			input:  goldInput(series(32, 0.5, 6)),
			expect: "HOLD, low confidence",
		},
		{
			name:   "sharp recent crash after a long rise",
			input:  goldInput(append(series(28, 0.6, 75), series(42, -3.0, 12)...)),
			expect: "BUY or SELL, not a shrug",
		},
	}

	runner := &CLIRunner{Timeout: 3 * time.Minute}
	seen := map[string]int{}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			prompt := BuildPrompt(sc.input)
			result, err := runner.Run(context.Background(), prompt, "claude-opus-5")
			if err != nil {
				t.Fatalf("runner: %v", err)
			}
			if result.IsError {
				t.Fatalf("CLI reported an error: %s", result.Result)
			}

			verdicts, err := ParseVerdicts(result.Result, []string{"gold"})
			if err != nil {
				t.Fatalf("parse: %v\nraw: %s", err, result.Result)
			}
			v := verdicts["gold"]
			seen[v.Signal]++

			fmt.Printf("\n--- %s\n    expected roughly: %s\n    got: %s (confidence %.2f)\n    %s\n",
				sc.name, sc.expect, v.Signal, v.Confidence, v.Reasoning)
		})
	}

	fmt.Printf("\n=== verdict spread across %d scenarios: %v\n", len(scenarios), seen)

	// The narrow question this probe exists to answer. One verdict for
	// every scenario means the prompt, not the market, is deciding.
	if len(seen) == 1 {
		for signal := range seen {
			t.Errorf("every scenario returned %s — the model is not discriminating between them", signal)
		}
	}
}
