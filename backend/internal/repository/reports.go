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
