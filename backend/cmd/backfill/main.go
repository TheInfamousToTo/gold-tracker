// Command backfill loads historical daily prices into gold_prices and
// silver_prices for the days the live feed never recorded.
//
// It is a dry run unless --apply is given, only ever inserts dates that
// have no row (ON CONFLICT DO NOTHING, so a live-feed row is never
// overwritten), and writes straight to the database so the API's
// new-price hook does not fire an AI run for prices years old.
//
// It reads the same DB_* environment as the API, so the intended way to
// run it is inside the API pod:
//
//	kubectl -n gold-tracker exec deploy/gold-tracker-api -- ./backfill
//	kubectl -n gold-tracker exec deploy/gold-tracker-api -- ./backfill --apply
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/backfill"
	"github.com/TheInfamousToTo/gold-tracker/backend/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type metal struct {
	name, table, column, source string
}

var metals = map[string]metal{
	"gold":   {"gold", "gold_prices", "price_per_gram_24k", "https://stooq.com/q/d/l/?s=xauusd&i=d"},
	"silver": {"silver", "silver_prices", "price_per_gram_999", "https://stooq.com/q/d/l/?s=xagusd&i=d"},
}

func main() {
	which := flag.String("metal", "both", "gold, silver or both")
	years := flag.Int("years", 10, "how many years back to fill")
	apply := flag.Bool("apply", false, "write the rows (default is a dry run)")
	factor := flag.Float64("factor", 0, "USD/g -> BHD/g multiplier; overrides the check against existing rows")
	goldCSV := flag.String("gold-csv", "", "path or URL of a gold Date,Close CSV in USD/oz (default: Stooq XAUUSD)")
	silverCSV := flag.String("silver-csv", "", "path or URL of a silver Date,Close CSV in USD/oz (default: Stooq XAGUSD)")
	flag.Parse()

	_ = godotenv.Load()
	repo, err := repository.NewPostgresRepository()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer repo.Close()

	sources := map[string]string{"gold": *goldCSV, "silver": *silverCSV}
	names := []string{"gold", "silver"}
	if *which != "both" {
		if _, ok := metals[*which]; !ok {
			log.Fatalf("--metal must be gold, silver or both")
		}
		names = []string{*which}
	}
	since := time.Now().AddDate(-*years, 0, 0).Format("2006-01-02")

	ctx := context.Background()
	failed := false
	for _, n := range names {
		m := metals[n]
		src := sources[n]
		if src == "" {
			src = m.source
		}
		if err := run(ctx, repo.Pool, m, src, since, *factor, *apply); err != nil {
			log.Printf("%s: %v", m.name, err)
			failed = true
		}
	}
	if !*apply {
		fmt.Println("\nDry run: nothing written. Re-run with --apply to insert.")
	}
	if failed {
		os.Exit(1)
	}
}

func run(ctx context.Context, pool *pgxpool.Pool, m metal, src, since string, override float64, apply bool) error {
	body, err := open(src)
	if err != nil {
		return err
	}
	defer body.Close()
	quotes, err := backfill.ParseCSV(body)
	if err != nil {
		return fmt.Errorf("%s: %w", src, err)
	}

	stored, err := existing(ctx, pool, m)
	if err != nil {
		return err
	}
	measured, overlap := backfill.MeasureFactor(quotes, stored)
	f, err := backfill.ChooseFactor(measured, overlap, override)
	if err != nil {
		return err
	}
	rows := backfill.Missing(quotes, stored, since, f)

	fmt.Printf("%s: %d CSV days (%s .. %s), %d rows already stored\n",
		m.name, len(quotes), quotes[0].Date, quotes[len(quotes)-1].Date, len(stored))
	if overlap > 0 {
		fmt.Printf("%s: existing rows imply %.5f BHD per USD over %d shared days; using %.5f\n", m.name, measured, overlap, f)
	} else {
		fmt.Printf("%s: no shared days; using --factor %.5f\n", m.name, f)
	}
	if len(rows) == 0 {
		fmt.Printf("%s: nothing missing since %s\n", m.name, since)
		return nil
	}
	fmt.Printf("%s: %d missing days from %s to %s, e.g. %s = %.3f BHD/g\n",
		m.name, len(rows), rows[0].Date, rows[len(rows)-1].Date, rows[len(rows)-1].Date, rows[len(rows)-1].Close)
	if !apply {
		return nil
	}

	tag := "backfill:" + sourceName(src)
	batch := &pgx.Batch{}
	for _, r := range rows {
		batch.Queue(fmt.Sprintf(
			`INSERT INTO %s (price_date, %s, source) VALUES ($1, $2, $3) ON CONFLICT (price_date) DO NOTHING`,
			m.table, m.column), r.Date, r.Close, tag)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	results := tx.SendBatch(ctx, batch)
	inserted := int64(0)
	for range rows {
		tag, err := results.Exec()
		if err != nil {
			results.Close()
			return err
		}
		inserted += tag.RowsAffected()
	}
	if err := results.Close(); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Printf("%s: inserted %d rows tagged %q\n", m.name, inserted, "backfill:"+sourceName(src))
	return nil
}

func existing(ctx context.Context, pool *pgxpool.Pool, m metal) (map[string]float64, error) {
	rows, err := pool.Query(ctx, fmt.Sprintf(`SELECT price_date, %s::float8 FROM %s`, m.column, m.table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var d time.Time
		var v float64
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
		out[d.Format("2006-01-02")] = v
	}
	return out, rows.Err()
}

func open(src string) (io.ReadCloser, error) {
	if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
		return os.Open(src)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(src)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w (download it elsewhere and pass the file with --gold-csv/--silver-csv)", src, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("fetching %s: HTTP %d", src, resp.StatusCode)
	}
	return resp.Body, nil
}

func sourceName(src string) string {
	if strings.Contains(src, "stooq.com") {
		return "stooq"
	}
	return "csv"
}
