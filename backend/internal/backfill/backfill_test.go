package backfill

import (
	"strings"
	"testing"
)

const stooq = `Date,Open,High,Low,Close
2026-10-05,2650,2660,2640,2655.5
2026-10-02,2600,2610,2590,2601.25
2026-10-06,2655,2670,2650,2662
bad-date,1,1,1,1
2026-10-07,2662,2680,2660,n/d
`

func TestParseCSVSortsAndSkipsBadRows(t *testing.T) {
	q, err := ParseCSV(strings.NewReader(stooq))
	if err != nil {
		t.Fatal(err)
	}
	if len(q) != 3 {
		t.Fatalf("got %d rows, want 3", len(q))
	}
	if q[0].Date != "2026-10-02" || q[2].Date != "2026-10-06" || q[0].Close != 2601.25 {
		t.Fatalf("unexpected rows %+v", q)
	}
}

// A rate-limited source answers with a page, not a CSV. That must stop
// the run rather than parse as zero rows of history.
func TestParseCSVRejectsNonCSV(t *testing.T) {
	if _, err := ParseCSV(strings.NewReader("Exceeded the daily hits limit")); err == nil {
		t.Fatal("expected an error for a non-CSV body")
	}
}

func TestMeasureFactorUsesOverlapMedian(t *testing.T) {
	quotes := []Quote{
		{"2026-10-01", GramsPerTroyOunce * 100},
		{"2026-10-02", GramsPerTroyOunce * 100},
		{"2026-10-03", GramsPerTroyOunce * 100},
		{"2026-10-04", GramsPerTroyOunce * 100}, // not stored
	}
	stored := map[string]float64{
		"2026-10-01": 37.6,
		"2026-10-02": 37.7,
		"2026-10-03": 99, // an outlier the median ignores
	}
	f, n := MeasureFactor(quotes, stored)
	if n != 3 || f != 0.377 {
		t.Fatalf("got factor %v over %d days", f, n)
	}
}

func TestChooseFactor(t *testing.T) {
	if f, err := ChooseFactor(0.3765, 20, 0); err != nil || f != BHDPerUSD {
		t.Fatalf("near-peg: got %v, %v", f, err)
	}
	if _, err := ChooseFactor(0.40, 20, 0); err == nil {
		t.Fatal("expected refusal far from the peg")
	}
	if _, err := ChooseFactor(0, 2, 0); err == nil {
		t.Fatal("expected refusal with too little overlap")
	}
	if f, _ := ChooseFactor(0, 0, 0.376); f != 0.376 {
		t.Fatal("override should win")
	}
}

func TestMissingSkipsStoredAndOldDays(t *testing.T) {
	quotes := []Quote{
		{"2020-01-01", GramsPerTroyOunce * 100},
		{"2026-10-01", GramsPerTroyOunce * 100},
		{"2026-10-02", GramsPerTroyOunce * 100},
	}
	stored := map[string]float64{"2026-10-02": 37.6}
	got := Missing(quotes, stored, "2021-01-01", BHDPerUSD)
	if len(got) != 1 || got[0].Date != "2026-10-01" || got[0].Close != 37.6 {
		t.Fatalf("got %+v", got)
	}
}
