package report

import (
	"reflect"
	"testing"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

func TestSureDates(t *testing.T) {
	var s []point
	for d := mustDate("2026-09-01"); !d.After(mustDate("2026-11-25")); d = d.AddDate(0, 0, 1) {
		s = append(s, point{d.Format("2006-01-02"), 40})
	}
	all := sureDates(s, "2026-09-20", "")
	// 2026-09-20 is a Sunday: the end of its ISO week, so it is kept.
	if all[0] != "2026-09-20" || all[1] != "2026-09-27" {
		t.Fatalf("weekly history starts %v", all[:3])
	}
	if all[len(all)-1] != "2026-11-25" || all[len(all)-2] != "2026-11-24" {
		t.Fatalf("recent days not daily: %v", all[len(all)-3:])
	}
	if got := sureDates(s, "2026-09-20", "2026-11-23"); !reflect.DeepEqual(got, []string{"2026-11-24", "2026-11-25"}) {
		t.Fatalf("resume = %v", got)
	}
	// Already synced: only the latest date, re-posted for same-day purchases.
	if got := sureDates(s, "2026-09-20", "2026-11-25"); !reflect.DeepEqual(got, []string{"2026-11-25"}) {
		t.Fatalf("up to date = %v", got)
	}
}

func TestMetalValue(t *testing.T) {
	s := []point{{"2026-11-01", 40}, {"2026-11-10", 50}}
	now21 := 50 * 21.0 / 24
	items := []model.PortfolioItem{
		{MetalType: "gold", PurityKarat: f(24), WeightGrams: 10, PurchaseDate: "2026-10-01"},
		{MetalType: "gold", PurityKarat: f(21), WeightGrams: 8, PurchaseDate: "2026-11-05", CurrentPricePerGram: &now21},
		{MetalType: "silver", PurityFineness: f(999), WeightGrams: 100, PurchaseDate: "2026-10-01"},
	}
	if v := metalValue(items, "gold", s, "2026-11-03"); v != 400 {
		t.Fatalf("before the ring: %v", v)
	}
	if v := metalValue(items, "gold", s, "2026-11-10"); v != 500+8*now21 {
		t.Fatalf("after the ring: %v", v)
	}
	// Bought after the last price arrived: still counted on the latest date.
	items = append(items, model.PortfolioItem{MetalType: "gold", PurityKarat: f(24), WeightGrams: 1, PurchaseDate: "2026-11-12"})
	if v := metalValue(items, "gold", s, "2026-11-10"); v != 550+8*now21 {
		t.Fatalf("same-day purchase missing: %v", v)
	}
}
