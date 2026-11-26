package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
	"github.com/jackc/pgx/v5"
)

const reportColumns = "month, created_at, data, analysis, model, error, discord_sent_at"

func scanReport(row pgx.Row) (model.MonthlyReport, error) {
	var r model.MonthlyReport
	var month time.Time
	var raw []byte
	if err := row.Scan(&month, &r.CreatedAt, &raw, &r.Analysis, &r.Model, &r.Error, &r.DiscordSentAt); err != nil {
		return r, err
	}
	r.Month = month.Format("2006-01")
	return r, json.Unmarshal(raw, &r.Data)
}

func (r *PostgresRepository) ListReports(ctx context.Context, limit int) ([]model.MonthlyReport, error) {
	rows, err := r.Pool.Query(ctx, "SELECT "+reportColumns+" FROM monthly_reports ORDER BY month DESC LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.MonthlyReport{}
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	return out, rows.Err()
}

// ReportExists reports whether a report for the YYYY-MM month is saved.
func (r *PostgresRepository) ReportExists(ctx context.Context, month string) (bool, error) {
	var one int
	err := r.Pool.QueryRow(ctx, "SELECT 1 FROM monthly_reports WHERE month = ($1 || '-01')::date", month).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// SaveReport inserts or replaces the month's report.
func (r *PostgresRepository) SaveReport(ctx context.Context, rep model.MonthlyReport) error {
	raw, err := json.Marshal(rep.Data)
	if err != nil {
		return err
	}
	_, err = r.Pool.Exec(ctx, `
		INSERT INTO monthly_reports (month, created_at, data, analysis, model, error, discord_sent_at)
		VALUES (($1 || '-01')::date, now(), $2, $3, $4, $5, $6)
		ON CONFLICT (month) DO UPDATE SET
			created_at = now(), data = EXCLUDED.data, analysis = EXCLUDED.analysis,
			model = EXCLUDED.model, error = EXCLUDED.error, discord_sent_at = EXCLUDED.discord_sent_at`,
		rep.Month, raw, rep.Analysis, rep.Model, rep.Error, rep.DiscordSentAt)
	return err
}

// LastPriceAlert returns the date and price of the last drop alert sent
// for the metal; ok is false when none has been sent.
func (r *PostgresRepository) LastPriceAlert(ctx context.Context, metal string) (string, float64, bool, error) {
	var on time.Time
	var price float64
	err := r.Pool.QueryRow(ctx, "SELECT alerted_on, price::float8 FROM price_alerts WHERE metal = $1", metal).Scan(&on, &price)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, err
	}
	return on.Format("2006-01-02"), price, true, nil
}

func (r *PostgresRepository) SavePriceAlert(ctx context.Context, metal, date string, price float64) error {
	_, err := r.Pool.Exec(ctx, `INSERT INTO price_alerts (metal, alerted_on, price) VALUES ($1, $2::date, $3)
		ON CONFLICT (metal) DO UPDATE SET alerted_on = EXCLUDED.alerted_on, price = EXCLUDED.price`, metal, date, price)
	return err
}

// SureSyncedThrough is the last price date posted to Sure for the metal.
func (r *PostgresRepository) SureSyncedThrough(ctx context.Context, metal string) (string, bool, error) {
	var d time.Time
	err := r.Pool.QueryRow(ctx, "SELECT synced_through FROM sure_sync WHERE metal = $1", metal).Scan(&d)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return d.Format("2006-01-02"), true, nil
}

func (r *PostgresRepository) SaveSureSynced(ctx context.Context, metal, date string) error {
	_, err := r.Pool.Exec(ctx, `INSERT INTO sure_sync (metal, synced_through) VALUES ($1, $2::date)
		ON CONFLICT (metal) DO UPDATE SET synced_through = EXCLUDED.synced_through`, metal, date)
	return err
}
