package model

import "time"

// MonthlyReport is one month's portfolio report: the computed figures,
// the model's written analysis, and whether it reached Discord.
type MonthlyReport struct {
	Month         string     `json:"month"` // YYYY-MM
	CreatedAt     time.Time  `json:"created_at"`
	Data          ReportData `json:"data"`
	Analysis      *string    `json:"analysis"`
	Model         *string    `json:"model"`
	Error         *string    `json:"error"`
	DiscordSentAt *time.Time `json:"discord_sent_at"`
}

// ReportData is every figure in a report. It is stored as JSON so a
// report keeps showing what was true when it was written.
type ReportData struct {
	Month       string          `json:"month"`
	GeneratedAt string          `json:"generated_at"`
	Items       []ReportItem    `json:"items"`
	Totals      ReportTotals    `json:"totals"`
	Metals      []MetalMonth    `json:"metals"`
	Purchases   []ReportItem    `json:"purchases"` // bought during the month
	Stocks      []StockPosition `json:"stocks,omitempty"`
	BudgetBHD   float64         `json:"budget_bhd"`
	SpentBHD    float64         `json:"spent_bhd"`
}

// ReportItem is one holding. MonthGain is how much its value moved
// during the report month (from the month's opening price, or from the
// purchase price if it was bought that month); nil when there was no
// price to measure against.
type ReportItem struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Metal       string   `json:"metal"`
	Purity      string   `json:"purity"`
	PurchasedOn string   `json:"purchased_on"`
	WeightGrams float64  `json:"weight_grams"`
	PaidTotal   float64  `json:"paid_total"`
	PaidPerGram float64  `json:"paid_per_gram"`
	NowPerGram  *float64 `json:"now_per_gram"`
	Value       *float64 `json:"value"`
	Gain        *float64 `json:"gain"`
	GainPct     *float64 `json:"gain_pct"`
	MonthGain   *float64 `json:"month_gain"`
}

type ReportTotals struct {
	Paid      float64 `json:"paid"`
	Value     float64 `json:"value"`
	Gain      float64 `json:"gain"`
	GainPct   float64 `json:"gain_pct"`
	MonthGain float64 `json:"month_gain"`
}

// MetalMonth is one metal's pure-metal price over the report month.
type MetalMonth struct {
	Metal     string  `json:"metal"`
	Open      float64 `json:"open"`
	Close     float64 `json:"close"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	ChangePct float64 `json:"change_pct"`
	Latest    float64 `json:"latest"`
	Days      int     `json:"days"`
}
