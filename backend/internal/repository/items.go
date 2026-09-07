package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// itemColumns is the column list every item query returns, in the
// order scanItem reads them.
const itemColumns = `id, purchase_date, item_name, metal_type, purity_karat, purity_fineness,
	weight_grams, price_paid_total, price_per_gram_paid, vendor, notes, created_at`

// rowScanner is satisfied by both pgx.Rows and pgx.Row, so the three
// item queries share one scan.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanItem(row rowScanner) (model.GoldItem, error) {
	var item model.GoldItem
	var purchaseDate time.Time
	err := row.Scan(
		&item.ID,
		&purchaseDate,
		&item.ItemName,
		&item.MetalType,
		&item.PurityKarat,
		&item.PurityFineness,
		&item.WeightGrams,
		&item.PricePaidTotal,
		&item.PricePerGramPaid,
		&item.Vendor,
		&item.Notes,
		&item.CreatedAt,
	)
	if err != nil {
		return item, err
	}
	item.PurchaseDate = purchaseDate.Format("2006-01-02")
	return item, nil
}

func (r *PostgresRepository) GetItems(ctx context.Context) ([]model.GoldItem, error) {
	rows, err := r.Pool.Query(ctx,
		"SELECT "+itemColumns+" FROM gold_items ORDER BY purchase_date DESC, id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.GoldItem
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) CreateItem(ctx context.Context, item model.GoldItem) (model.GoldItem, error) {
	row := r.Pool.QueryRow(ctx,
		`INSERT INTO gold_items (purchase_date, item_name, metal_type, purity_karat, purity_fineness, weight_grams, price_paid_total, vendor, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING `+itemColumns,
		item.PurchaseDate, item.ItemName, item.MetalType, item.PurityKarat, item.PurityFineness,
		item.WeightGrams, item.PricePaidTotal, item.Vendor, item.Notes,
	)
	return scanItem(row)
}

func (r *PostgresRepository) UpdateItem(ctx context.Context, id int, item model.GoldItem) (model.GoldItem, error) {
	row := r.Pool.QueryRow(ctx,
		`UPDATE gold_items SET purchase_date = $1, item_name = $2, metal_type = $3, purity_karat = $4,
		        purity_fineness = $5, weight_grams = $6, price_paid_total = $7, vendor = $8, notes = $9
		 WHERE id = $10
		 RETURNING `+itemColumns,
		item.PurchaseDate, item.ItemName, item.MetalType, item.PurityKarat, item.PurityFineness,
		item.WeightGrams, item.PricePaidTotal, item.Vendor, item.Notes, id,
	)
	return scanItem(row)
}

func (r *PostgresRepository) DeleteItem(ctx context.Context, id int) error {
	commandTag, err := r.Pool.Exec(ctx, "DELETE FROM gold_items WHERE id = $1", id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("item not found")
	}
	return nil
}
