package repository

import (
	"context"
	"time"

	"github.com/your-username/gold-tracker/backend/internal/model"
)

func (r *PostgresRepository) GetPortfolioSummary(ctx context.Context) (model.PortfolioSummary, error) {
	rows, err := r.Pool.Query(ctx, "SELECT * FROM v_portfolio_summary ORDER BY purchase_date DESC, id DESC")
	if err != nil {
		return model.PortfolioSummary{}, err
	}
	defer rows.Close()

	var summary model.PortfolioSummary
	var totalPaid, totalValue float64

	for rows.Next() {
		var item model.PortfolioItem
		var purchaseDate time.Time
		var latestPriceDate *time.Time
		err := rows.Scan(
			&item.ID,
			&item.ItemName,
			&purchaseDate,
			&item.PurityKarat,
			&item.WeightGrams,
			&item.PricePaidTotal,
			&item.PricePerGramPaid,
			&latestPriceDate,
			&item.CurrentPricePerGram,
			&item.CurrentValue,
			&item.GainLoss,
			&item.GainLossPct,
		)
		if err != nil {
			return model.PortfolioSummary{}, err
		}
		item.PurchaseDate = purchaseDate.Format("2006-01-02")
		if latestPriceDate != nil {
			d := latestPriceDate.Format("2006-01-02")
			item.LatestPriceDate = &d
		}
		summary.Items = append(summary.Items, item)

		totalPaid += item.PricePaidTotal
		if item.CurrentValue != nil {
			totalValue += *item.CurrentValue
		}
	}

	summary.Totals.TotalPaid = totalPaid
	summary.Totals.TotalValue = totalValue
	summary.Totals.TotalGainLoss = totalValue - totalPaid
	if totalPaid > 0 {
		summary.Totals.TotalGainLossPct = (summary.Totals.TotalGainLoss / totalPaid) * 100
	}
	
	summary.HasPriceData = len(summary.Items) > 0 && summary.Items[0].LatestPriceDate != nil

	return summary, nil
}
