package repository

import (
	"context"
	"errors"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetSignals(ctx context.Context, limit int) ([]model.SignalLog, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id, signal_date, signal_type, reasoning, price_at_signal, sent_to_discord, model, source FROM signals_log ORDER BY signal_date DESC LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	signals := []model.SignalLog{}
	for rows.Next() {
		var s model.SignalLog
		err := rows.Scan(
			&s.ID,
			&s.SignalDate,
			&s.SignalType,
			&s.Reasoning,
			&s.PriceAtSignal,
			&s.SentToDiscord,
			&s.Model,
			&s.Source,
		)
		if err != nil {
			return nil, err
		}
		signals = append(signals, s)
	}
	return signals, nil
}

func (r *PostgresRepository) CreateSignal(ctx context.Context, s model.SignalLog) (model.SignalLog, error) {
	var newSignal model.SignalLog
	err := r.Pool.QueryRow(ctx,
		`INSERT INTO signals_log (signal_type, reasoning, price_at_signal, model, source)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, signal_date, signal_type, reasoning, price_at_signal, sent_to_discord, model, source`,
		s.SignalType, s.Reasoning, s.PriceAtSignal, s.Model, s.Source,
	).Scan(
		&newSignal.ID,
		&newSignal.SignalDate,
		&newSignal.SignalType,
		&newSignal.Reasoning,
		&newSignal.PriceAtSignal,
		&newSignal.SentToDiscord,
		&newSignal.Model,
		&newSignal.Source,
	)
	return newSignal, err
}

// GetLatestSignal returns the most recent signal for the given source,
// or nil when that source has produced none yet. A missing row is not
// an error — callers use it to decide whether the daily cap is due.
func (r *PostgresRepository) GetLatestSignal(ctx context.Context, source string) (*model.SignalLog, error) {
	var s model.SignalLog
	err := r.Pool.QueryRow(ctx,
		`SELECT id, signal_date, signal_type, reasoning, price_at_signal, sent_to_discord, model, source
		 FROM signals_log WHERE source = $1 ORDER BY signal_date DESC LIMIT 1`,
		source,
	).Scan(
		&s.ID,
		&s.SignalDate,
		&s.SignalType,
		&s.Reasoning,
		&s.PriceAtSignal,
		&s.SentToDiscord,
		&s.Model,
		&s.Source,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}
