package model

import (
	"time"
)

type GoldItem struct {
	ID                int       `json:"id"`
	PurchaseDate      string    `json:"purchase_date"`
	ItemName          string    `json:"item_name"`
	MetalType         string    `json:"metal_type"`
	PurityKarat       float64   `json:"purity_karat"`
	WeightGrams       float64   `json:"weight_grams"`
	PricePaidTotal    float64   `json:"price_paid_total"`
	PricePerGramPaid  float64   `json:"price_per_gram_paid"`
	Vendor            *string   `json:"vendor"`
	Notes             *string   `json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
}

type GoldPrice struct {
	ID              int       `json:"id"`
	PriceDate       string    `json:"price_date"`
	PricePerGram24k float64   `json:"price_per_gram_24k"`
	PricePerGram22k float64   `json:"price_per_gram_22k"`
	PricePerGram21k float64   `json:"price_per_gram_21k"`
	PricePerGram18k float64   `json:"price_per_gram_18k"`
	Source          string    `json:"source"`
	CreatedAt       time.Time `json:"created_at"`
}

type SignalLog struct {
	ID            int       `json:"id"`
	SignalDate    time.Time `json:"signal_date"`
	SignalType    string    `json:"signal_type"`
	Reasoning     *string   `json:"reasoning"`
	PriceAtSignal *float64  `json:"price_at_signal"`
	SentToDiscord bool      `json:"sent_to_discord"`
	Model         *string   `json:"model"`
	Source        string    `json:"source"`
}

type PortfolioItem struct {
	ID                  int      `json:"id"`
	ItemName            string   `json:"item_name"`
	PurchaseDate        string   `json:"purchase_date"`
	PurityKarat         float64  `json:"purity_karat"`
	WeightGrams         float64  `json:"weight_grams"`
	PricePaidTotal      float64  `json:"price_paid_total"`
	PricePerGramPaid    float64  `json:"price_per_gram_paid"`
	LatestPriceDate     *string  `json:"latest_price_date"`
	CurrentPricePerGram *float64 `json:"current_price_per_gram"`
	CurrentValue        *float64 `json:"current_value"`
	GainLoss            *float64 `json:"gain_loss"`
	GainLossPct         *float64 `json:"gain_loss_pct"`
}

type PortfolioSummary struct {
	Items         []PortfolioItem `json:"items"`
	Totals        PortfolioTotals `json:"totals"`
	HasPriceData  bool            `json:"has_price_data"`
}

type PortfolioTotals struct {
	TotalPaid        float64 `json:"total_paid"`
	TotalValue       float64 `json:"total_value"`
	TotalGainLoss    float64 `json:"total_gain_loss"`
	TotalGainLossPct float64 `json:"total_gain_loss_pct"`
}
