package model

import (
    "errors"
    "math"
    "strings"
    "time"
)

// Broker fees are configured by the owner, not guessed from a provider's
// changing tariff. Fixed and minimum fees use the trade's quote currency.
type StockBroker struct {
    ID int `json:"id"`
    Name string `json:"name"`
    BuyPercent float64 `json:"buy_percent"`
    SellPercent float64 `json:"sell_percent"`
    FixedFee float64 `json:"fixed_fee"`
    MinimumFee float64 `json:"minimum_fee"`
    TaxPercent float64 `json:"tax_percent"`
}

func finiteNonnegative(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 }

func (b StockBroker) Validate() error {
    if strings.TrimSpace(b.Name) == "" || len(b.Name) > 120 {
        return errors.New("broker name is required (max 120 characters)")
    }
    if !finiteNonnegative(b.BuyPercent) || b.BuyPercent > 100 ||
        !finiteNonnegative(b.SellPercent) || b.SellPercent > 100 ||
        !finiteNonnegative(b.FixedFee) || !finiteNonnegative(b.MinimumFee) ||
        !finiteNonnegative(b.TaxPercent) || b.TaxPercent > 100 {
        return errors.New("broker fees must be non-negative and percentages at most 100")
    }
    return nil
}

func roundBHD(v float64) float64 { return math.Round(v*1000) / 1000 }

// Fee returns the charge in quote currency. Commission is percentage
// plus fixed, floored to the broker's minimum; tax applies to that fee.
func (b StockBroker) Fee(side string, notional float64) float64 {
    rate := b.BuyPercent
    if side == "sell" { rate = b.SellPercent }
    commission := math.Max(notional*rate/100+b.FixedFee, b.MinimumFee)
    return commission * (1+b.TaxPercent/100)
}

type StockTradeInput struct {
    Symbol string `json:"symbol"`
    AssetName string `json:"asset_name"`
    Side string `json:"side"`
    TradeDate string `json:"trade_date"`
    Shares float64 `json:"shares"`
    UnitPrice float64 `json:"unit_price"`
    Currency string `json:"currency"`
    FXToBHD float64 `json:"fx_to_bhd"`
    BrokerID *int `json:"broker_id"`
    Notes string `json:"notes"`
}

func (t *StockTradeInput) Validate() error {
    t.Symbol = strings.ToUpper(strings.TrimSpace(t.Symbol))
    t.AssetName = strings.TrimSpace(t.AssetName)
    t.Currency = strings.ToUpper(strings.TrimSpace(t.Currency))
    if len(t.Symbol) == 0 || len(t.Symbol) > 16 {
        return errors.New("symbol must be 1–16 characters")
    }
    for _, c := range t.Symbol {
        if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '-') {
            return errors.New("symbol must contain only letters, digits, dots, or hyphens")
        }
    }
    if t.AssetName == "" || len(t.AssetName) > 120 {
        return errors.New("asset_name is required (max 120 characters)")
    }
    if t.Side != "buy" && t.Side != "sell" { return errors.New("side must be buy or sell") }
    tradeDate, err := time.Parse("2006-01-02", t.TradeDate)
    if err != nil {
        return errors.New("trade_date must be YYYY-MM-DD")
    }
    // Trades must be entered in date order, so one future-dated trade
    // would block every real entry for that symbol until its date. One
    // day of slack covers Bahrain (UTC+3) being ahead of the server.
    if tradeDate.After(time.Now().UTC().AddDate(0, 0, 1)) {
        return errors.New("trade_date cannot be in the future")
    }
    if !finiteNonnegative(t.Shares) || t.Shares == 0 ||
       !finiteNonnegative(t.UnitPrice) || t.UnitPrice == 0 ||
       !finiteNonnegative(t.FXToBHD) || t.FXToBHD == 0 {
        return errors.New("shares, unit_price, and fx_to_bhd must be positive")
    }
    if len(t.Currency) != 3 {
        return errors.New("currency must be a three-letter ISO currency code")
    }
    for _, c := range t.Currency {
        if c < 'A' || c > 'Z' { return errors.New("currency must contain only letters") }
    }
    if t.Currency == "BHD" && t.FXToBHD != 1 {
        return errors.New("fx_to_bhd must equal 1 for BHD trades")
    }
    if t.BrokerID != nil && *t.BrokerID <= 0 { return errors.New("invalid broker_id") }
    if len(t.Notes) > 2000 { return errors.New("notes too long") }
    return nil
}

type StockTrade struct {
    ID int64 `json:"id"`
    StockTradeInput
    BrokerName *string `json:"broker_name"`
    NotionalBHD float64 `json:"notional_bhd"`
    FeesBHD float64 `json:"fees_bhd"`
    TotalBHD float64 `json:"total_bhd"`
    RealizedPNLBHD float64 `json:"realized_pnl_bhd"`
}

type StockPosition struct {
    Symbol string `json:"symbol"`
    AssetName string `json:"asset_name"`
    Currency string `json:"currency"`
    Shares float64 `json:"shares"`
    CostBasisBHD float64 `json:"cost_basis_bhd"`
    AvgCostBHD float64 `json:"avg_cost_bhd"`
    RealizedPNLBHD float64 `json:"realized_pnl_bhd"`
    LatestPrice *float64 `json:"latest_price"`
    LatestFXToBHD *float64 `json:"latest_fx_to_bhd"`
    LatestPriceDate *string `json:"latest_price_date"`
    MarketValueBHD *float64 `json:"market_value_bhd"`
    UnrealizedPNLBHD *float64 `json:"unrealized_pnl_bhd"`
    // Estimated sale proceeds use the latest quote before any future broker fees.
}

type StockQuoteInput struct {
    Price float64 `json:"price"`
    FXToBHD float64 `json:"fx_to_bhd"`
    PriceDate string `json:"price_date"`
}

func (q StockQuoteInput) Validate() error {
    if !finiteNonnegative(q.Price) || q.Price == 0 ||
        !finiteNonnegative(q.FXToBHD) || q.FXToBHD == 0 {
        return errors.New("price and fx_to_bhd must be positive")
    }
    if _, err := time.Parse("2006-01-02", q.PriceDate); err != nil {
        return errors.New("price_date must be YYYY-MM-DD")
    }
    return nil
}

// Stock accounting uses a moving weighted-average basis including
// purchase fees. Sales remove proportional basis and subtract sale fees.
func ApplyTrade(shares, basis, realized, quantity, notional, fees float64, side string) (float64, float64, float64, error) {
    if side == "buy" {
        return shares+quantity, roundBHD(basis+notional+fees), realized, nil
    }
    if side != "sell" || quantity > shares+1e-9 || shares <= 0 {
        return shares, basis, realized, errors.New("insufficient shares to sell")
    }
    allocated := basis * quantity/shares
    remaining := shares - quantity
    if math.Abs(remaining) < 1e-9 { remaining = 0 }
    if remaining == 0 { allocated = basis }
    newBasis := roundBHD(basis - allocated)
    if remaining == 0 { newBasis = 0 }
    return remaining, newBasis, roundBHD(realized+notional-fees-allocated), nil
}
