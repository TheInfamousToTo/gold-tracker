package repository

import (
	"context"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

func (r *PostgresRepository) GetSignals(ctx context.Context, limit int) ([]model.SignalLog, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id, signal_date, signal_type, reasoning, price_at_signal, sent_to_discord FROM signals_log ORDER BY signal_date DESC LIMIT $1", limit)
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
		)
		if err != nil {
			return nil, err
		}
		signals = append(signals, s)
	}
	return signals, nil
}
