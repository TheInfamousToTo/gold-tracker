package model

import (
	"time"
)

// Metals the tracker understands. Gold is traded in karat and silver
// only in millesimal fineness, so an item carries whichever purity
// column its metal uses and leaves the other nil.
const (
	MetalGold   = "gold"
	MetalSilver = "silver"
)

// IsKnownMetal reports whether a metal is one the schema's CHECK
// constraint will accept, so a bad value fails as a 400 rather than a
// constraint violation surfacing as a 500.
func IsKnownMetal(metal string) bool {
	return metal == MetalGold || metal == MetalSilver
}

type GoldItem struct {
	ID               int       `json:"id"`
	PurchaseDate     string    `json:"purchase_date"`
	ItemName         string    `json:"item_name"`
	MetalType        string    `json:"metal_type"`
	PurityKarat      *float64  `json:"purity_karat"`
	PurityFineness   *float64  `json:"purity_fineness"`
	WeightGrams      float64   `json:"weight_grams"`
	PricePaidTotal   float64   `json:"price_paid_total"`
	PricePerGramPaid float64   `json:"price_per_gram_paid"`
	Vendor           *string   `json:"vendor"`
	Notes            *string   `json:"notes"`
	CreatedAt        time.Time `json:"created_at"`
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

// SilverPrice mirrors GoldPrice. Silver is quoted independently of
// gold and a day may carry one without the other, so it keeps its own
// table rather than sharing a row.
type SilverPrice struct {
	ID              int       `json:"id"`
	PriceDate       string    `json:"price_date"`
	PricePerGram999 float64   `json:"price_per_gram_999"`
	PricePerGram925 float64   `json:"price_per_gram_925"`
	PricePerGram900 float64   `json:"price_per_gram_900"`
	Source          string    `json:"source"`
	CreatedAt       time.Time `json:"created_at"`
}

type SignalLog struct {
	ID            int       `json:"id"`
	SignalDate    time.Time `json:"signal_date"`
	Metal         string    `json:"metal"`
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
	MetalType           string   `json:"metal_type"`
	PurityKarat         *float64 `json:"purity_karat"`
	PurityFineness      *float64 `json:"purity_fineness"`
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
