package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
	"github.com/jackc/pgx/v5"
)

// signalColumns is shared by every read so the positional scan in
// scanSignal cannot drift from the SELECT list.
const signalColumns = `id, signal_date, metal, signal_type, reasoning, price_at_signal,
	sent_to_discord, model, source, kind, confidence, horizon_days, key_factors,
	amount_bhd, amount_grams, levels, news, news_checked, deviates,
	suggested_action, plan_text, baseline`

// JSONB columns are scanned as raw bytes and decoded here: a NULL then
// simply leaves the field empty, where scanning straight into a slice
// would fail on rows written before the column existed.
func scanSignal(row pgx.Row) (model.SignalLog, error) {
	var s model.SignalLog
	var factors, levels, news, baseline []byte
	err := row.Scan(
		&s.ID, &s.SignalDate, &s.Metal, &s.SignalType, &s.Reasoning, &s.PriceAtSignal,
		&s.SentToDiscord, &s.Model, &s.Source, &s.Kind, &s.Confidence, &s.HorizonDays, &factors,
		&s.AmountBHD, &s.AmountGrams, &levels, &news, &s.NewsChecked, &s.Deviates,
		&s.SuggestedAction, &s.PlanText, &baseline,
	)
	if err != nil {
		return s, err
	}
	if err := decodeJSON(factors, &s.KeyFactors); err != nil {
		return s, fmt.Errorf("signal %d key_factors: %w", s.ID, err)
	}
	if len(levels) > 0 && string(levels) != "null" {
		s.Levels = &model.Levels{}
		if err := json.Unmarshal(levels, s.Levels); err != nil {
			return s, fmt.Errorf("signal %d levels: %w", s.ID, err)
		}
	}
	if err := decodeJSON(news, &s.News); err != nil {
		return s, fmt.Errorf("signal %d news: %w", s.ID, err)
	}
	if len(baseline) > 0 {
		s.Baseline = json.RawMessage(baseline)
	}
	return s, nil
}

func decodeJSON(raw []byte, into any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, into)
}

// encodeJSON returns nil for an empty value so the column stays NULL
// rather than holding "null" or "[]".
func encodeJSON(v any, empty bool) ([]byte, error) {
	if empty {
		return nil, nil
	}
	return json.Marshal(v)
}

func (r *PostgresRepository) GetSignals(ctx context.Context, limit int) ([]model.SignalLog, error) {
	rows, err := r.Pool.Query(ctx,
		"SELECT "+signalColumns+" FROM signals_log ORDER BY signal_date DESC, id DESC LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	signals := []model.SignalLog{}
	for rows.Next() {
		s, err := scanSignal(rows)
		if err != nil {
			return nil, err
		}
		signals = append(signals, s)
	}
	return signals, rows.Err()
}

func (r *PostgresRepository) CreateSignal(ctx context.Context, s model.SignalLog) (model.SignalLog, error) {
	if s.Kind == "" {
		s.Kind = model.SignalKindAnalysis
	}
	factors, err := encodeJSON(s.KeyFactors, len(s.KeyFactors) == 0)
	if err != nil {
		return model.SignalLog{}, err
	}
	levels, err := encodeJSON(s.Levels, s.Levels == nil)
	if err != nil {
		return model.SignalLog{}, err
	}
	news, err := encodeJSON(s.News, len(s.News) == 0)
	if err != nil {
		return model.SignalLog{}, err
	}
	var baseline []byte
	if len(s.Baseline) > 0 {
		baseline = s.Baseline
	}

	return scanSignal(r.Pool.QueryRow(ctx,
		`INSERT INTO signals_log (metal, signal_type, reasoning, price_at_signal, model, source,
			kind, confidence, horizon_days, key_factors, amount_bhd, amount_grams, levels, news,
			news_checked, deviates, suggested_action, plan_text, baseline)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		 RETURNING `+signalColumns,
		s.Metal, s.SignalType, s.Reasoning, s.PriceAtSignal, s.Model, s.Source,
		s.Kind, s.Confidence, s.HorizonDays, factors, s.AmountBHD, s.AmountGrams, levels, news,
		s.NewsChecked, s.Deviates, s.SuggestedAction, s.PlanText, baseline,
	))
}

// GetLatestSignal returns the most recent analysis for the given source,
// or nil when that source has produced none yet. A missing row is not
// an error — callers use it to decide whether the daily cap is due.
// Reviews are excluded: asking about a plan must not reset the timer
// on the daily analysis.
func (r *PostgresRepository) GetLatestSignal(ctx context.Context, source string) (*model.SignalLog, error) {
	s, err := scanSignal(r.Pool.QueryRow(ctx,
		"SELECT "+signalColumns+` FROM signals_log
		 WHERE source = $1 AND kind = 'analysis' ORDER BY signal_date DESC LIMIT 1`,
		source,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// GetAdvisorSettings reads the single settings row. Migration 0003
// inserts it, so a missing row means the schema is behind; the
// defaults are returned rather than failing the analysis.
func (r *PostgresRepository) GetAdvisorSettings(ctx context.Context) (model.AdvisorSettings, error) {
	var s model.AdvisorSettings
	err := r.Pool.QueryRow(ctx,
		`SELECT monthly_budget_bhd, reserve_bhd, target_gold_pct, spread_pct_gold,
		        spread_pct_silver, min_fee_bhd, stop_loss_pct, news_enabled, updated_at
		 FROM ai_settings WHERE id = 1`,
	).Scan(&s.MonthlyBudgetBHD, &s.ReserveBHD, &s.TargetGoldPct, &s.SpreadPctGold,
		&s.SpreadPctSilver, &s.MinFeeBHD, &s.StopLossPct, &s.NewsEnabled, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.DefaultAdvisorSettings(), nil
	}
	return s, err
}

func (r *PostgresRepository) UpdateAdvisorSettings(ctx context.Context, s model.AdvisorSettings) (model.AdvisorSettings, error) {
	var out model.AdvisorSettings
	err := r.Pool.QueryRow(ctx,
		`INSERT INTO ai_settings (id, monthly_budget_bhd, reserve_bhd, target_gold_pct,
			spread_pct_gold, spread_pct_silver, min_fee_bhd, stop_loss_pct, news_enabled, updated_at)
		 VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, now())
		 ON CONFLICT (id) DO UPDATE SET
			monthly_budget_bhd = EXCLUDED.monthly_budget_bhd,
			reserve_bhd = EXCLUDED.reserve_bhd,
			target_gold_pct = EXCLUDED.target_gold_pct,
			spread_pct_gold = EXCLUDED.spread_pct_gold,
			spread_pct_silver = EXCLUDED.spread_pct_silver,
			min_fee_bhd = EXCLUDED.min_fee_bhd,
			stop_loss_pct = EXCLUDED.stop_loss_pct,
			news_enabled = EXCLUDED.news_enabled,
			updated_at = now()
		 RETURNING monthly_budget_bhd, reserve_bhd, target_gold_pct, spread_pct_gold,
		           spread_pct_silver, min_fee_bhd, stop_loss_pct, news_enabled, updated_at`,
		s.MonthlyBudgetBHD, s.ReserveBHD, s.TargetGoldPct, s.SpreadPctGold,
		s.SpreadPctSilver, s.MinFeeBHD, s.StopLossPct, s.NewsEnabled,
	).Scan(&out.MonthlyBudgetBHD, &out.ReserveBHD, &out.TargetGoldPct, &out.SpreadPctGold,
		&out.SpreadPctSilver, &out.MinFeeBHD, &out.StopLossPct, &out.NewsEnabled, &out.UpdatedAt)
	return out, err
}
