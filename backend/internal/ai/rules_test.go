package ai

import (
	"testing"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// state builds a gold state from indicators alone; the rules never
// look at the raw series.
func state(metal string, ind Indicators, pos Position) MetalState {
	if ind.Count == 0 {
		ind.Count = 300
	}
	return MetalState{Metal: metal, Indicators: ind, Position: pos}
}

func flat(z float64) Indicators {
	return Indicators{Latest: 50, SMA50: 49, SMA200: 48, StdDev200: 2, Z200: z, TrendUp: true}
}

func settingsWith(f func(*model.AdvisorSettings)) model.AdvisorSettings {
	s := model.DefaultAdvisorSettings()
	s.MonthlyBudgetBHD = 100
	if f != nil {
		f(&s)
	}
	return s
}

func budget(monthly, spent, reserve float64) Budget {
	rem := monthly - spent
	if rem < 0 {
		rem = 0
	}
	return Budget{Monthly: monthly, Spent: spent, Remaining: rem, Reserve: reserve}
}

func TestBaselineBuysTheMonthlyShareInNormalTimes(t *testing.T) {
	b := baselineFor(state("gold", flat(0), Position{}), 1, settingsWith(nil), budget(100, 0, 0), 0)
	if b.Action != ActionBuy || b.AmountBHD != 100 || b.Multiplier != 1 {
		t.Fatalf("got %+v, want BUY 100 at 1x", b)
	}
	// Grams at the price actually paid: rate plus the 1% spread.
	near(t, "grams", b.AmountGrams, round3v(100/(50*1.01)), 1e-9)
}

func TestBaselineBuysExtraOnADeepDip(t *testing.T) {
	b := baselineFor(state("gold", flat(-2), Position{}), 1, settingsWith(nil), budget(100, 0, 200), 0)
	// 1.5 x 100, plus 25% of the 200 reserve.
	if b.Action != ActionBuy || b.AmountBHD != 200 || b.ReserveDraw != 50 {
		t.Fatalf("got %+v, want BUY 200 drawing 50 from the reserve", b)
	}
}

func TestBaselineBuysHalfWhenStretched(t *testing.T) {
	b := baselineFor(state("gold", flat(2), Position{}), 1, settingsWith(nil), budget(100, 0, 0), 0)
	if b.Action != ActionBuy || b.AmountBHD != 50 || b.ToReserve != 50 {
		t.Fatalf("got %+v, want BUY 50 with 50 parked", b)
	}
}

func TestBaselineHoldsWhenTheBuyIsTooSmallForTheFee(t *testing.T) {
	s := settingsWith(func(s *model.AdvisorSettings) { s.MonthlyBudgetBHD = 15; s.MinFeeBHD = 2 })
	b := baselineFor(state("gold", flat(0), Position{}), 1, s, budget(15, 0, 0), 0)
	if b.Action != ActionHold {
		t.Fatalf("got %+v, want HOLD (15 BHD < 10 x 2 BHD fee)", b)
	}
}

func TestBaselineHoldsWithoutBudget(t *testing.T) {
	s := settingsWith(func(s *model.AdvisorSettings) { s.MonthlyBudgetBHD = 0 })
	if b := baselineFor(state("gold", flat(0), Position{}), 1, s, budget(0, 0, 0), 0); b.Action != ActionHold {
		t.Errorf("no budget: got %+v, want HOLD", b)
	}
	if b := baselineFor(state("gold", flat(0), Position{}), 1, settingsWith(nil), budget(100, 100, 0), 0); b.Action != ActionHold {
		t.Errorf("budget used: got %+v, want HOLD", b)
	}
}

func brokenTrend() Indicators {
	return Indicators{Latest: 40, SMA50: 44, SMA200: 47, StdDev200: 3, Z200: -2.3, TrendUp: false}
}

func TestBaselineCutsALossOnlyWhenTrendAndStopAgree(t *testing.T) {
	losing := Position{Held: true, FineGrams: 20, Paid: 1000, Value: 800, NetValue: 792, NetPLPct: -20.8, Share: 1}

	b := baselineFor(state("gold", brokenTrend(), losing), 1, settingsWith(nil), budget(100, 0, 0), 800)
	if b.Action != ActionCutLoss || b.AmountBHD != 792 || b.AmountGrams != 20 {
		t.Fatalf("got %+v, want CUT_LOSS of the whole 792 BHD / 20 g", b)
	}

	// The same loss in an intact uptrend is a dip to buy, not a cut.
	intact := brokenTrend()
	intact.SMA50 = 48
	if b := baselineFor(state("gold", intact, losing), 1, settingsWith(nil), budget(100, 0, 0), 800); b.Action != ActionBuy {
		t.Errorf("intact trend: got %+v, want BUY", b)
	}

	// A broken trend with a loss inside the stop is held through.
	small := losing
	small.NetPLPct = -10
	if b := baselineFor(state("gold", brokenTrend(), small), 1, settingsWith(nil), budget(0, 0, 0), 800); b.Action == ActionCutLoss {
		t.Errorf("loss inside the stop: got CUT_LOSS")
	}
}

func TestBaselineTrimsOnlyWhenStretchedAndOverweight(t *testing.T) {
	ind := Indicators{Latest: 60, SMA50: 55, SMA200: 45, StdDev200: 5, Z200: 3, TrendUp: true, Mom252: 50, HasMom252: true}
	heavy := Position{Held: true, FineGrams: 30, Paid: 900, Value: 1800, NetValue: 1782, NetPLPct: 98, Share: 0.95}

	b := baselineFor(state("gold", ind, heavy), 1, settingsWith(nil), budget(100, 0, 0), 2000)
	// Back from 95% to the 80% target of 2000, net of the 1% spread.
	if b.Action != ActionSell {
		t.Fatalf("got %+v, want SELL", b)
	}
	near(t, "sell amount", b.AmountBHD, round2v(0.15*2000*0.99), 1e-9)

	balanced := heavy
	balanced.Share = 0.85
	if b := baselineFor(state("gold", ind, balanced), 1, settingsWith(nil), budget(100, 0, 0), 2000); b.Action == ActionSell {
		t.Errorf("not overweight: got SELL")
	}
}

func TestBaselineLevels(t *testing.T) {
	pos := Position{Held: true, FineGrams: 20, Paid: 900, NetPLPct: 5}
	b := baselineFor(state("gold", flat(0), pos), 1, settingsWith(nil), budget(100, 0, 0), 0)
	near(t, "buy_more_below", *b.BuyMoreBelow, 48-1.5*2, 1e-9)
	near(t, "cut_loss_below", *b.CutLossBelow, round3v(900*0.85/(20*0.99)), 1e-9)
}

func TestSplitSharesLeansToTheUnderweightMetalAndTheRatio(t *testing.T) {
	states := []MetalState{
		state("gold", flat(0), Position{Held: true, Share: 1}),
		state("silver", flat(0), Position{}),
	}
	s := settingsWith(nil)

	shares := splitShares(states, s, RatioStats{})
	near(t, "gold share, all-gold portfolio", shares["gold"], 0.6, 1e-9)

	shares = splitShares(states, s, RatioStats{Available: true, Percentile: 0.9})
	near(t, "gold share, cheap silver", shares["gold"], 0.4, 1e-9)
	near(t, "silver share, cheap silver", shares["silver"], 0.6, 1e-9)
}

func TestSplitSharesGivesEverythingToTheOnlyPricedMetal(t *testing.T) {
	states := []MetalState{state("gold", flat(0), Position{})}
	if got := splitShares(states, settingsWith(nil), RatioStats{}); got["gold"] != 1 {
		t.Errorf("gold share = %v, want 1 with no silver prices", got["gold"])
	}
}
