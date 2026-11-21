// Package report builds the monthly portfolio report: per-item figures,
// the metals' month, an AI-written analysis, and a Discord digest.
package report

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// point is one day's pure-metal price per gram (24K gold, 999 silver).
type point struct {
	Date  string
	Price float64
}

// Inputs is everything Build reads. Price series may be in any order.
type Inputs struct {
	Month    string // YYYY-MM
	Now      time.Time
	Items    []model.PortfolioItem
	Gold     []model.GoldPrice
	Silver   []model.SilverPrice
	Stocks   []model.StockPosition
	Settings model.AdvisorSettings
}

// monthBounds returns the month's first and last day as YYYY-MM-DD.
func monthBounds(month string) (string, string, error) {
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return "", "", fmt.Errorf("month must be YYYY-MM: %w", err)
	}
	end := start.AddDate(0, 1, -1)
	return start.Format("2006-01-02"), end.Format("2006-01-02"), nil
}

// PreviousMonth is the last complete calendar month before t.
func PreviousMonth(t time.Time) string {
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	return first.AddDate(0, -1, 0).Format("2006-01")
}

func series(in Inputs) map[string][]point {
	gold := make([]point, 0, len(in.Gold))
	for _, p := range in.Gold {
		gold = append(gold, point{p.PriceDate, p.PricePerGram24k})
	}
	silver := make([]point, 0, len(in.Silver))
	for _, p := range in.Silver {
		silver = append(silver, point{p.PriceDate, p.PricePerGram999})
	}
	for _, s := range [][]point{gold, silver} {
		sort.Slice(s, func(i, j int) bool { return s[i].Date < s[j].Date })
	}
	return map[string][]point{model.MetalGold: gold, model.MetalSilver: silver}
}

// onOrBefore is the last price dated on or before date.
func onOrBefore(s []point, date string) (float64, bool) {
	i := sort.Search(len(s), func(i int) bool { return s[i].Date > date })
	if i == 0 {
		return 0, false
	}
	return s[i-1].Price, true
}

func metalOf(item model.PortfolioItem) string {
	if item.MetalType == "" {
		return model.MetalGold
	}
	return item.MetalType
}

// purityFactor converts the pure-metal price to this item's price per
// gram. It is read back from the portfolio view when possible, so the
// report always agrees with the holdings table.
func purityFactor(item model.PortfolioItem, latestPure float64) float64 {
	if item.CurrentPricePerGram != nil && latestPure > 0 {
		return *item.CurrentPricePerGram / latestPure
	}
	if metalOf(item) == model.MetalSilver {
		if item.PurityFineness != nil {
			return *item.PurityFineness / 999
		}
		return 0
	}
	if item.PurityKarat != nil {
		return *item.PurityKarat / 24
	}
	return 0
}

func purityLabel(item model.PortfolioItem) string {
	if metalOf(item) == model.MetalSilver && item.PurityFineness != nil {
		return fmt.Sprintf("%g", *item.PurityFineness)
	}
	if item.PurityKarat != nil {
		return fmt.Sprintf("%gK", *item.PurityKarat)
	}
	return "?"
}

func ptr(v float64) *float64 { return &v }

func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}

// Build computes the report figures for in.Month.
func Build(in Inputs) (model.ReportData, error) {
	start, end, err := monthBounds(in.Month)
	if err != nil {
		return model.ReportData{}, err
	}
	dayBefore := mustDate(start).AddDate(0, 0, -1).Format("2006-01-02")
	all := series(in)

	data := model.ReportData{
		Month:       in.Month,
		GeneratedAt: in.Now.Format(time.RFC3339),
		Items:       []model.ReportItem{},
		Purchases:   []model.ReportItem{},
		BudgetBHD:   in.Settings.MonthlyBudgetBHD,
	}

	for _, metal := range []string{model.MetalGold, model.MetalSilver} {
		s := all[metal]
		if len(s) == 0 {
			continue
		}
		mm := model.MetalMonth{Metal: metal, Latest: s[len(s)-1].Price, Low: math.Inf(1)}
		for _, p := range s {
			if p.Date >= start && p.Date <= end {
				if mm.Days == 0 {
					mm.Open = p.Price // fallback when nothing precedes the month
				}
				mm.Days++
				mm.High = math.Max(mm.High, p.Price)
				mm.Low = math.Min(mm.Low, p.Price)
				mm.Close = p.Price
			}
		}
		if mm.Days == 0 {
			continue
		}
		if prev, ok := onOrBefore(s, dayBefore); ok {
			mm.Open = prev
		}
		if mm.Open > 0 {
			mm.ChangePct = round((mm.Close/mm.Open-1)*100, 2)
		}
		data.Metals = append(data.Metals, mm)
	}

	for _, item := range in.Items {
		metal := metalOf(item)
		s := all[metal]
		var latest float64
		if len(s) > 0 {
			latest = s[len(s)-1].Price
		}
		factor := purityFactor(item, latest)

		row := model.ReportItem{
			ID: item.ID, Name: item.ItemName, Metal: metal, Purity: purityLabel(item),
			PurchasedOn: item.PurchaseDate, WeightGrams: item.WeightGrams,
			PaidTotal: item.PricePaidTotal, PaidPerGram: round(item.PricePerGramPaid, 3),
		}
		if latest > 0 && factor > 0 {
			now := latest * factor
			value := now * item.WeightGrams
			row.NowPerGram = ptr(round(now, 3))
			row.Value = ptr(round(value, 3))
			row.Gain = ptr(round(value-item.PricePaidTotal, 3))
			if item.PricePaidTotal > 0 {
				row.GainPct = ptr(round((value/item.PricePaidTotal-1)*100, 2))
			}
			data.Totals.Value += value
		}

		// Month gain: from the month's opening price (or the purchase
		// price if bought during the month) to the month's close.
		if item.PurchaseDate <= end && factor > 0 {
			closePure, okClose := onOrBefore(s, end)
			ref, okRef := item.PricePerGramPaid, item.PurchaseDate >= start
			if !okRef {
				var open float64
				open, okRef = onOrBefore(s, dayBefore)
				ref = open * factor
			}
			if okClose && okRef {
				g := item.WeightGrams * (closePure*factor - ref)
				row.MonthGain = ptr(round(g, 3))
				data.Totals.MonthGain += g
			}
		}

		data.Totals.Paid += item.PricePaidTotal
		data.Items = append(data.Items, row)
		if item.PurchaseDate >= start && item.PurchaseDate <= end {
			data.Purchases = append(data.Purchases, row)
			data.SpentBHD += item.PricePaidTotal
		}
	}

	data.Totals.Gain = data.Totals.Value - data.Totals.Paid
	if data.Totals.Paid > 0 {
		data.Totals.GainPct = round(data.Totals.Gain/data.Totals.Paid*100, 2)
	}
	data.Totals.Paid = round(data.Totals.Paid, 3)
	data.Totals.Value = round(data.Totals.Value, 3)
	data.Totals.Gain = round(data.Totals.Gain, 3)
	data.Totals.MonthGain = round(data.Totals.MonthGain, 3)
	data.SpentBHD = round(data.SpentBHD, 3)

	for _, p := range in.Stocks {
		if p.Shares > 0 {
			data.Stocks = append(data.Stocks, p)
		}
	}
	return data, nil
}

func mustDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}
