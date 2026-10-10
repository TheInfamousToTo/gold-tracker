package ai

import (
	"math"
	"sort"
	"strings"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// Indicators are the long-run figures the rules and the model reason
// from. They are computed here, over the whole history, because a model
// doing this arithmetic in its head over thousands of rows drifts.
//
// Windows are counted in observations (trading days), not calendar
// days: 200 observations is the conventional "200-day average".
type Indicators struct {
	Count      int
	Latest     float64
	LatestDate string

	SMA50  float64
	SMA200 float64
	// StdDev200 is the standard deviation of the last 200 closes, and
	// Z200 is how many of those the latest close sits from SMA200 — the
	// single "how stretched is price" figure the rules key on.
	StdDev200 float64
	Z200      float64
	// PctVsSMA200 is the latest close against SMA200, in percent.
	PctVsSMA200 float64
	TrendUp     bool

	// Percentile of the latest close within the last 252 and 756
	// closes (about one and three years), 0..1.
	Pctile252 float64
	Pctile756 float64
	// DrawdownPct is the fall from the series high, in percent (≤ 0).
	DrawdownPct float64
	SeriesHigh  float64

	// Momentum over the last 63, 126 and 252 observations, in percent.
	// Absent when the series is too short.
	Mom63, Mom126, Mom252          float64
	HasMom63, HasMom126, HasMom252 bool

	// AnnualVolPct is the annualised volatility of daily log returns
	// over the last 252 observations, in percent.
	AnnualVolPct float64
}

// Sparse reports a series too short for the 200-observation figures
// to mean what their names say.
func (in Indicators) Sparse() bool { return in.Count < 200 }

// computeIndicators expects prices oldest first.
func computeIndicators(prices []PriceHistoryPoint) Indicators {
	var ind Indicators
	ind.Count = len(prices)
	if len(prices) == 0 {
		return ind
	}
	closes := make([]float64, len(prices))
	for i, p := range prices {
		closes[i] = p.PricePerGram
	}
	latest := closes[len(closes)-1]
	ind.Latest = latest
	ind.LatestDate = prices[len(prices)-1].Date

	ind.SMA50 = mean(tail(closes, 50))
	w200 := tail(closes, 200)
	ind.SMA200 = mean(w200)
	ind.StdDev200 = stddev(w200)
	if ind.StdDev200 > 0 {
		ind.Z200 = (latest - ind.SMA200) / ind.StdDev200
	}
	if ind.SMA200 > 0 {
		ind.PctVsSMA200 = (latest/ind.SMA200 - 1) * 100
	}
	ind.TrendUp = ind.SMA50 > ind.SMA200

	ind.Pctile252 = percentileOf(tail(closes, 252), latest)
	ind.Pctile756 = percentileOf(tail(closes, 756), latest)

	for _, c := range closes {
		ind.SeriesHigh = math.Max(ind.SeriesHigh, c)
	}
	if ind.SeriesHigh > 0 {
		ind.DrawdownPct = (latest/ind.SeriesHigh - 1) * 100
	}

	ind.Mom63, ind.HasMom63 = momentum(closes, 63)
	ind.Mom126, ind.HasMom126 = momentum(closes, 126)
	ind.Mom252, ind.HasMom252 = momentum(closes, 252)

	ind.AnnualVolPct = annualVol(tail(closes, 253)) * 100
	return ind
}

func tail(xs []float64, n int) []float64 {
	if len(xs) <= n {
		return xs
	}
	return xs[len(xs)-n:]
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

// stddev is the population standard deviation: the window is the
// whole population being described, not a sample of a larger one.
func stddev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	m := mean(xs)
	var ss float64
	for _, x := range xs {
		ss += (x - m) * (x - m)
	}
	return math.Sqrt(ss / float64(len(xs)))
}

// percentileOf is the fraction of xs at or below v.
func percentileOf(xs []float64, v float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var n int
	for _, x := range xs {
		if x <= v {
			n++
		}
	}
	return float64(n) / float64(len(xs))
}

func momentum(closes []float64, n int) (float64, bool) {
	if len(closes) <= n {
		return 0, false
	}
	first := closes[len(closes)-n-1]
	if first == 0 {
		return 0, false
	}
	return (closes[len(closes)-1]/first - 1) * 100, true
}

func annualVol(closes []float64) float64 {
	if len(closes) < 3 {
		return 0
	}
	rets := make([]float64, 0, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		if closes[i-1] > 0 && closes[i] > 0 {
			rets = append(rets, math.Log(closes[i]/closes[i-1]))
		}
	}
	return stddev(rets) * math.Sqrt(252)
}

// RatioStats describes the gold/silver ratio: how many grams of fine
// silver one gram of fine gold buys. Per gram in BHD it is the same
// number as the familiar USD-per-ounce ratio, because both legs carry
// the same conversion.
type RatioStats struct {
	Available bool
	Latest    float64
	// Percentile of the latest ratio within the shared history, 0..1.
	// High means silver is cheap against gold.
	Percentile float64
	Days       int
}

// computeRatio pairs the two series on shared dates. Both inputs are
// oldest first.
func computeRatio(gold, silver []PriceHistoryPoint) RatioStats {
	silverByDate := make(map[string]float64, len(silver))
	for _, p := range silver {
		silverByDate[p.Date] = p.PricePerGram
	}
	var ratios []float64
	for _, g := range gold {
		if s, ok := silverByDate[g.Date]; ok && s > 0 {
			ratios = append(ratios, g.PricePerGram/s)
		}
	}
	if len(ratios) < 30 {
		return RatioStats{Days: len(ratios)}
	}
	latest := ratios[len(ratios)-1]
	return RatioStats{
		Available:  true,
		Latest:     latest,
		Percentile: percentileOf(ratios, latest),
		Days:       len(ratios),
	}
}

// Position is what the owner holds of one metal, valued at what it
// would actually fetch: the portfolio view's value less the spread.
type Position struct {
	FineGrams float64
	Paid      float64
	Value     float64
	NetValue  float64
	// NetPLPct is the gain or loss after the spread, in percent.
	NetPLPct float64
	// Share is this metal's fraction of the whole portfolio's value.
	Share float64
	Held  bool
}

// computePositions sums items per metal. Items with no current value
// (no price yet) count toward what was paid but not toward value.
func computePositions(items []model.PortfolioItem, settings model.AdvisorSettings) map[string]Position {
	out := map[string]Position{}
	var total float64
	for _, item := range items {
		metal := item.MetalType
		if metal == "" {
			metal = model.MetalGold
		}
		p := out[metal]
		p.FineGrams += item.WeightGrams * fineFraction(metal, item)
		p.Paid += item.PricePaidTotal
		if item.CurrentValue != nil {
			p.Value += *item.CurrentValue
			total += *item.CurrentValue
		}
		p.Held = p.Held || item.WeightGrams > 0
		out[metal] = p
	}
	for metal, p := range out {
		p.NetValue = p.Value * (1 - settings.SpreadFraction(metal))
		if p.Paid > 0 {
			p.NetPLPct = (p.NetValue/p.Paid - 1) * 100
		}
		if total > 0 {
			p.Share = p.Value / total
		}
		out[metal] = p
	}
	return out
}

func fineFraction(metal string, item model.PortfolioItem) float64 {
	if metal == model.MetalSilver {
		if item.PurityFineness == nil {
			return 0
		}
		return *item.PurityFineness / 1000
	}
	if item.PurityKarat == nil {
		return 0
	}
	return *item.PurityKarat / 24
}

// spentThisMonth sums purchases dated in the given YYYY-MM month, so
// the advisor knows how much of the monthly budget is already used.
func spentThisMonth(items []model.PortfolioItem, month string) float64 {
	var sum float64
	for _, item := range items {
		if strings.HasPrefix(item.PurchaseDate, month) {
			sum += item.PricePaidTotal
		}
	}
	return sum
}

// monthlyCloses keeps the last close of each calendar month, which is
// enough for the model to see a decade's shape in ~120 rows.
func monthlyCloses(prices []PriceHistoryPoint) []PriceHistoryPoint {
	var out []PriceHistoryPoint
	for i, p := range prices {
		last := i == len(prices)-1
		if last || prices[i+1].Date[:7] != p.Date[:7] {
			out = append(out, p)
		}
	}
	return out
}

// sortedOldestFirst copies and sorts a series by date.
func sortedOldestFirst(points []PriceHistoryPoint) []PriceHistoryPoint {
	out := append([]PriceHistoryPoint(nil), points...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}
