package ai

import (
	"math"
	"testing"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// linear builds n daily closes from start, adding step each day.
func linear(start, step float64, n int) []PriceHistoryPoint {
	out := make([]PriceHistoryPoint, n)
	day := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range out {
		out[i] = PriceHistoryPoint{Date: day.AddDate(0, 0, i).Format("2006-01-02"), PricePerGram: start + step*float64(i)}
	}
	return out
}

func near(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %v, want %v (±%v)", name, got, want, tol)
	}
}

func TestIndicatorsOnAStraightLine(t *testing.T) {
	// 300 closes 1, 2, ... 300: every figure has a closed form.
	ind := computeIndicators(linear(1, 1, 300))

	near(t, "Latest", ind.Latest, 300, 0)
	near(t, "SMA50", ind.SMA50, 275.5, 1e-9)
	near(t, "SMA200", ind.SMA200, 200.5, 1e-9)
	// Population σ of 200 consecutive integers is sqrt((200²-1)/12).
	near(t, "StdDev200", ind.StdDev200, math.Sqrt((200*200-1)/12.0), 1e-9)
	near(t, "Z200", ind.Z200, 99.5/math.Sqrt((200*200-1)/12.0), 1e-9)
	if !ind.TrendUp {
		t.Error("TrendUp = false on a rising line")
	}
	near(t, "Pctile252", ind.Pctile252, 1, 0)
	near(t, "DrawdownPct", ind.DrawdownPct, 0, 0)
	near(t, "Mom63", ind.Mom63, (300.0/237-1)*100, 1e-9)
	if !ind.HasMom252 || ind.Sparse() {
		t.Errorf("HasMom252=%v Sparse=%v on 300 closes", ind.HasMom252, ind.Sparse())
	}
}

func TestIndicatorsReportDrawdownAndShortSeries(t *testing.T) {
	prices := append(linear(100, 1, 50), linear(149, -2, 20)...)
	ind := computeIndicators(prices)
	near(t, "SeriesHigh", ind.SeriesHigh, 149, 0)
	near(t, "DrawdownPct", ind.DrawdownPct, (111.0/149-1)*100, 1e-9)
	if !ind.Sparse() || ind.HasMom252 || ind.HasMom126 {
		t.Errorf("Sparse=%v HasMom126=%v HasMom252=%v on 70 closes", ind.Sparse(), ind.HasMom126, ind.HasMom252)
	}
}

func TestIndicatorsOnEmptyInput(t *testing.T) {
	if ind := computeIndicators(nil); ind.Count != 0 || ind.Latest != 0 {
		t.Errorf("got %+v, want zero value", ind)
	}
}

func TestRatioPairsOnSharedDatesOnly(t *testing.T) {
	gold := linear(50, 0, 40)
	silver := linear(0.5, 0, 40)[5:] // silver lacks the first five days
	r := computeRatio(gold, silver)
	if !r.Available || r.Days != 35 {
		t.Fatalf("got %+v, want 35 shared days", r)
	}
	near(t, "Latest", r.Latest, 100, 1e-9)
	if got := computeRatio(gold, silver[:10]); got.Available {
		t.Errorf("ratio over 10 days marked available: %+v", got)
	}
}

func TestPositionsNetOfSpread(t *testing.T) {
	k24, f999 := 24.0, 999.0
	gv, sv := 1000.0, 250.0
	items := []model.PortfolioItem{
		{MetalType: "gold", PurityKarat: &k24, WeightGrams: 20, PricePaidTotal: 900, CurrentValue: &gv},
		{MetalType: "silver", PurityFineness: &f999, WeightGrams: 300, PricePaidTotal: 300, CurrentValue: &sv},
	}
	s := model.DefaultAdvisorSettings()
	s.SpreadPctGold, s.SpreadPctSilver = 1, 2
	pos := computePositions(items, s)

	g := pos["gold"]
	near(t, "gold FineGrams", g.FineGrams, 20, 1e-9)
	near(t, "gold NetValue", g.NetValue, 990, 1e-9)
	near(t, "gold NetPLPct", g.NetPLPct, 10, 1e-9)
	near(t, "gold Share", g.Share, 0.8, 1e-9)
	sl := pos["silver"]
	near(t, "silver FineGrams", sl.FineGrams, 299.7, 1e-9)
	near(t, "silver NetPLPct", sl.NetPLPct, (245.0/300-1)*100, 1e-9)
}

func TestSpentThisMonth(t *testing.T) {
	items := []model.PortfolioItem{
		{PurchaseDate: "2026-10-01", PricePaidTotal: 20},
		{PurchaseDate: "2026-10-09", PricePaidTotal: 15},
		{PurchaseDate: "2026-09-30", PricePaidTotal: 99},
	}
	near(t, "spent", spentThisMonth(items, "2026-10"), 35, 0)
}

func TestMonthlyClosesKeepsTheLastCloseOfEachMonth(t *testing.T) {
	prices := linear(1, 1, 70) // 2020-01-01 .. 2020-03-10
	got := monthlyCloses(prices)
	want := []string{"2020-01-31", "2020-02-29", "2020-03-10"}
	if len(got) != len(want) {
		t.Fatalf("got %d closes, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Date != w {
			t.Errorf("close %d on %s, want %s", i, got[i].Date, w)
		}
	}
}
