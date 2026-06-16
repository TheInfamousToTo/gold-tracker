package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/your-username/gold-tracker/backend/internal/model"
)

func (r *PostgresRepository) GetItems(ctx context.Context) ([]model.GoldItem, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id, purchase_date, item_name, metal_type, purity_karat, weight_grams, price_paid_total, price_per_gram_paid, vendor, notes, created_at FROM gold_items ORDER BY purchase_date DESC, id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.GoldItem
	for rows.Next() {
		var item model.GoldItem
		var purchaseDate time.Time
		err := rows.Scan(
			&item.ID,
			&purchaseDate,
			&item.ItemName,
			&item.MetalType,
			&item.PurityKarat,
			&item.WeightGrams,
			&item.PricePaidTotal,
			&item.PricePerGramPaid,
			&item.Vendor,
			&item.Notes,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		item.PurchaseDate = purchaseDate.Format("2006-01-02")
		items = append(items, item)
	}
	return items, nil
}

func (r *PostgresRepository) CreateItem(ctx context.Context, item model.GoldItem) (model.GoldItem, error) {
	var newItem model.GoldItem
	var purchaseDate time.Time
	err := r.Pool.QueryRow(ctx, 
		`INSERT INTO gold_items (purchase_date, item_name, metal_type, purity_karat, weight_grams, price_paid_total, vendor, notes) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
		 RETURNING id, purchase_date, item_name, metal_type, purity_karat, weight_grams, price_paid_total, price_per_gram_paid, vendor, notes, created_at`,
		item.PurchaseDate, item.ItemName, item.MetalType, item.PurityKarat, item.WeightGrams, item.PricePaidTotal, item.Vendor, item.Notes,
	).Scan(
		&newItem.ID,
		&purchaseDate,
		&newItem.ItemName,
		&newItem.MetalType,
		&newItem.PurityKarat,
		&newItem.WeightGrams,
		&newItem.PricePaidTotal,
		&newItem.PricePerGramPaid,
		&newItem.Vendor,
		&newItem.Notes,
		&newItem.CreatedAt,
	)
	if err != nil {
		return newItem, err
	}
	newItem.PurchaseDate = purchaseDate.Format("2006-01-02")
	return newItem, nil
}

func (r *PostgresRepository) UpdateItem(ctx context.Context, id int, item model.GoldItem) (model.GoldItem, error) {
	var updatedItem model.GoldItem
	var purchaseDate time.Time
	err := r.Pool.QueryRow(ctx, 
		`UPDATE gold_items SET purchase_date = $1, item_name = $2, metal_type = $3, purity_karat = $4, weight_grams = $5, price_paid_total = $6, vendor = $7, notes = $8 
		 WHERE id = $9 
		 RETURNING id, purchase_date, item_name, metal_type, purity_karat, weight_grams, price_paid_total, price_per_gram_paid, vendor, notes, created_at`,
		item.PurchaseDate, item.ItemName, item.MetalType, item.PurityKarat, item.WeightGrams, item.PricePaidTotal, item.Vendor, item.Notes, id,
	).Scan(
		&updatedItem.ID,
		&purchaseDate,
		&updatedItem.ItemName,
		&updatedItem.MetalType,
		&updatedItem.PurityKarat,
		&updatedItem.WeightGrams,
		&updatedItem.PricePaidTotal,
		&updatedItem.PricePerGramPaid,
		&updatedItem.Vendor,
		&updatedItem.Notes,
		&updatedItem.CreatedAt,
	)
	if err != nil {
		return updatedItem, err
	}
	updatedItem.PurchaseDate = purchaseDate.Format("2006-01-02")
	return updatedItem, nil
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
