package model

import (
	"encoding/json"
	"errors"
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

	// Kind separates the scheduled or on-demand analysis from a review
	// of a plan the owner typed. A review's SignalType is its verdict
	// (GOOD_IDEA, ADJUST, BAD_IDEA) and SuggestedAction is what it
	// recommends doing instead, if anything.
	Kind            string     `json:"kind"`
	Confidence      *float64   `json:"confidence"`
	HorizonDays     *int       `json:"horizon_days"`
	KeyFactors      []string   `json:"key_factors"`
	AmountBHD       *float64   `json:"amount_bhd"`
	AmountGrams     *float64   `json:"amount_grams"`
	Levels          *Levels    `json:"levels"`
	News            []NewsItem `json:"news"`
	NewsChecked     bool       `json:"news_checked"`
	Deviates        bool       `json:"deviates"`
	SuggestedAction *string    `json:"suggested_action"`
	PlanText        *string    `json:"plan_text"`

	// Baseline is the rules engine's own answer, kept so a verdict that
	// departs from it can be compared against what the rules said.
	Baseline json.RawMessage `json:"baseline,omitempty"`
}

const (
	SignalKindAnalysis = "analysis"
	SignalKindReview   = "review"
)

// Levels are the prices at which the call would change: below
// BuyMoreBelow the dip is deep enough to buy extra, and below
// CutLossBelow the position has fallen past the owner's stop.
type Levels struct {
	BuyMoreBelow *float64 `json:"buy_more_below,omitempty"`
	CutLossBelow *float64 `json:"cut_loss_below,omitempty"`
}

// NewsItem is one article the model cited. Only items with a real
// http(s) URL are kept, so every claim can be opened and checked.
type NewsItem struct {
	Title  string `json:"title"`
	Source string `json:"source"`
	Date   string `json:"date"`
	URL    string `json:"url"`
	Impact string `json:"impact"`
}

// AdvisorSettings is what the advisor sizes against. There is one row.
type AdvisorSettings struct {
	MonthlyBudgetBHD float64    `json:"monthly_budget_bhd"`
	ReserveBHD       float64    `json:"reserve_bhd"`
	TargetGoldPct    float64    `json:"target_gold_pct"`
	SpreadPctGold    float64    `json:"spread_pct_gold"`
	SpreadPctSilver  float64    `json:"spread_pct_silver"`
	MinFeeBHD        float64    `json:"min_fee_bhd"`
	StopLossPct      float64    `json:"stop_loss_pct"`
	NewsEnabled      bool       `json:"news_enabled"`
	UpdatedAt        *time.Time `json:"updated_at"`
}

// DefaultAdvisorSettings matches the column defaults in migration 0003.
func DefaultAdvisorSettings() AdvisorSettings {
	return AdvisorSettings{
		TargetGoldPct:   80,
		SpreadPctGold:   1,
		SpreadPctSilver: 1,
		StopLossPct:     15,
		NewsEnabled:     true,
	}
}

// Validate bounds every field to a range that means something, so a
// typo cannot tell the advisor to size against a negative budget.
func (s AdvisorSettings) Validate() error {
	switch {
	case s.MonthlyBudgetBHD < 0 || s.MonthlyBudgetBHD > 1e6:
		return errors.New("monthly_budget_bhd must be between 0 and 1,000,000")
	case s.ReserveBHD < 0 || s.ReserveBHD > 1e7:
		return errors.New("reserve_bhd must be between 0 and 10,000,000")
	case s.TargetGoldPct < 0 || s.TargetGoldPct > 100:
		return errors.New("target_gold_pct must be between 0 and 100")
	case s.SpreadPctGold < 0 || s.SpreadPctGold > 20, s.SpreadPctSilver < 0 || s.SpreadPctSilver > 20:
		return errors.New("spreads must be between 0 and 20 percent")
	case s.MinFeeBHD < 0 || s.MinFeeBHD > 1000:
		return errors.New("min_fee_bhd must be between 0 and 1,000")
	case s.StopLossPct <= 0 || s.StopLossPct > 90:
		return errors.New("stop_loss_pct must be above 0 and at most 90")
	}
	return nil
}

// SpreadFraction is the one-way spread for a metal as a fraction.
func (s AdvisorSettings) SpreadFraction(metal string) float64 {
	if metal == MetalSilver {
		return s.SpreadPctSilver / 100
	}
	return s.SpreadPctGold / 100
}

// TargetShare is a metal's target fraction of the portfolio.
func (s AdvisorSettings) TargetShare(metal string) float64 {
	if metal == MetalSilver {
		return 1 - s.TargetGoldPct/100
	}
	return s.TargetGoldPct / 100
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
