package repository

import (
	"context"
	"time"

	"github.com/your-username/gold-tracker/backend/internal/model"
)

func (r *PostgresRepository) GetPrices(ctx context.Context, limit int) ([]model.GoldPrice, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id, price_date, price_per_gram_24k, price_per_gram_22k, price_per_gram_21k, price_per_gram_18k, source, created_at FROM gold_prices ORDER BY price_date DESC LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []model.GoldPrice
	for rows.Next() {
		var p model.GoldPrice
		var priceDate time.Time
		err := rows.Scan(
			&p.ID,
			&priceDate,
			&p.PricePerGram24k,
			&p.PricePerGram22k,
			&p.PricePerGram21k,
			&p.PricePerGram18k,
			&p.Source,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		p.PriceDate = priceDate.Format("2006-01-02")
		prices = append(prices, p)
	}
	return prices, nil
}

func (r *PostgresRepository) CreatePrice(ctx context.Context, p model.GoldPrice) (model.GoldPrice, error) {
	var newPrice model.GoldPrice
	var priceDate time.Time
	err := r.Pool.QueryRow(ctx, 
		`INSERT INTO gold_prices (price_date, price_per_gram_24k, source) 
		 VALUES ($1, $2, $3) 
		 ON CONFLICT (price_date) DO UPDATE 
		 SET price_per_gram_24k = EXCLUDED.price_per_gram_24k, 
		     source = EXCLUDED.source 
		 RETURNING id, price_date, price_per_gram_24k, price_per_gram_22k, price_per_gram_21k, price_per_gram_18k, source, created_at`,
		p.PriceDate, p.PricePerGram24k, p.Source,
	).Scan(
		&newPrice.ID,
		&priceDate,
		&newPrice.PricePerGram24k,
		&newPrice.PricePerGram22k,
		&newPrice.PricePerGram21k,
		&newPrice.PricePerGram18k,
		&newPrice.Source,
		&newPrice.CreatedAt,
	)
	if err != nil {
		return newPrice, err
	}
	newPrice.PriceDate = priceDate.Format("2006-01-02")
	return newPrice, nil
}
