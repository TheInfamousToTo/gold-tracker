// Package backfill fills the price tables with history the live feed
// never recorded. The live feed (n8n, GoldAPI) writes one row a day going
// forward; a chart window of years needs the years before it started.
//
// History comes from a bulk daily-close CSV in USD per troy ounce, which
// is converted to BHD per gram. The conversion factor is not assumed: it
// is measured against the rows the live feed already wrote, so the
// backfilled series joins the live one without a step at the seam.
package backfill

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GramsPerTroyOunce converts a per-ounce quote to per-gram.
const GramsPerTroyOunce = 31.1034768

// BHDPerUSD is the Bahraini dinar's fixed peg, unchanged since 2001.
const BHDPerUSD = 0.376

// PegTolerance is how far the measured factor may sit from the peg and
// still be treated as the peg. The live feed and a daily close are taken
// at different moments of the day, so they never agree exactly.
const PegTolerance = 0.005

// MinOverlap is the fewest shared days the factor is measured from.
const MinOverlap = 5

// Quote is one day's close in USD per troy ounce.
type Quote struct {
	Date  string // YYYY-MM-DD
	Close float64
}

// ParseCSV reads a daily series with a header row naming a "Date" and a
// "Close" column, in any order and any case. That is Stooq's layout and
// what most exports produce. Rows with an unparseable close are skipped;
// a file with no usable rows is an error, since it usually means the
// source answered with an HTML page or a rate-limit notice instead.
func ParseCSV(r io.Reader) ([]Quote, error) {
	rd := csv.NewReader(r)
	rd.FieldsPerRecord = -1
	header, err := rd.Read()
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	dateCol, closeCol := -1, -1
	for i, h := range header {
		switch strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, "\ufeff"))) {
		case "date":
			dateCol = i
		case "close", "price":
			closeCol = i
		}
	}
	if dateCol < 0 || closeCol < 0 {
		return nil, fmt.Errorf("no Date/Close columns in header %q (is this really a CSV?)", strings.Join(header, ","))
	}

	var out []Quote
	for {
		rec, err := rd.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(rec) <= dateCol || len(rec) <= closeCol {
			continue
		}
		day, err := time.Parse("2006-01-02", strings.TrimSpace(rec[dateCol]))
		if err != nil {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(rec[closeCol]), 64)
		if err != nil || v <= 0 {
			continue
		}
		out = append(out, Quote{Date: day.Format("2006-01-02"), Close: v})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no usable rows")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out, nil
}

// MeasureFactor returns the median of stored BHD/g over CSV USD/g across
// the days both series have, and how many days that was.
func MeasureFactor(quotes []Quote, stored map[string]float64) (float64, int) {
	var ratios []float64
	for _, q := range quotes {
		if bhd, ok := stored[q.Date]; ok && bhd > 0 {
			ratios = append(ratios, bhd/(q.Close/GramsPerTroyOunce))
		}
	}
	if len(ratios) == 0 {
		return 0, 0
	}
	sort.Float64s(ratios)
	mid := len(ratios) / 2
	if len(ratios)%2 == 1 {
		return ratios[mid], len(ratios)
	}
	return (ratios[mid-1] + ratios[mid]) / 2, len(ratios)
}

// ChooseFactor decides the USD/g -> BHD/g multiplier. An explicit
// override wins. Otherwise the measured factor must match the peg, in
// which case the exact peg is used; anything else means the live feed
// converts differently from what this tool assumes, and writing would
// put a visible step into the chart, so it refuses.
func ChooseFactor(measured float64, overlap int, override float64) (float64, error) {
	if override > 0 {
		return override, nil
	}
	if overlap < MinOverlap {
		return 0, fmt.Errorf("only %d day(s) overlap the existing rows (need %d) to check the USD->BHD factor; pass --factor %.3f to accept the peg", overlap, MinOverlap, BHDPerUSD)
	}
	if math.Abs(measured/BHDPerUSD-1) > PegTolerance {
		return 0, fmt.Errorf("existing rows imply a factor of %.5f, not the %.3f peg; check how the live feed converts, then pass --factor %.5f if that is intended", measured, BHDPerUSD, measured)
	}
	return BHDPerUSD, nil
}

// Missing returns the quotes on or after `since` whose date has no row
// yet, converted to BHD per gram and rounded to the 3 decimals the
// dinar is quoted in.
func Missing(quotes []Quote, stored map[string]float64, since string, factor float64) []Quote {
	var out []Quote
	for _, q := range quotes {
		if q.Date < since {
			continue
		}
		if _, ok := stored[q.Date]; ok {
			continue
		}
		perGram := q.Close / GramsPerTroyOunce * factor
		out = append(out, Quote{Date: q.Date, Close: math.Round(perGram*1000) / 1000})
	}
	return out
}
