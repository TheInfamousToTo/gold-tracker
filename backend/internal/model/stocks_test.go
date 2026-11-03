package model

import (
    "math"
    "testing"
    "time"
)

func TestBrokerFeeIncludesMinimumFixedAndTax(t *testing.T) {
    b := StockBroker{Name: "Test", BuyPercent: 0.5, SellPercent: 1,
        FixedFee: 0.5, MinimumFee: 2, TaxPercent: 10}
    if got := b.Fee("buy", 100); math.Abs(got-2.2) > 1e-9 {
        t.Errorf("buy fee = %v, want 2.2", got)
    }
    if got := b.Fee("sell", 1000); math.Abs(got-11.55) > 1e-9 {
        t.Errorf("sell fee = %v, want 11.55", got)
    }
}

func TestTradeValidation(t *testing.T) {
    valid := StockTradeInput{
        Symbol: "gld", AssetName: "Gold ETF", Side: "buy",
        TradeDate: "2026-11-03", Shares: 1.5,
        UnitPrice: 250, Currency: "USD", FXToBHD: 0.377,
    }
    if err := valid.Validate(); err != nil { t.Fatalf("valid trade: %v", err) }
    if valid.Symbol != "GLD" { t.Errorf("symbol = %s, want GLD", valid.Symbol) }
    invalid := valid
    invalid.UnitPrice = math.Inf(1)
    if err := invalid.Validate(); err == nil { t.Error("accepted infinite price") }
    invalid = valid
    invalid.FXToBHD = -1
    if err := invalid.Validate(); err == nil { t.Error("accepted negative exchange rate") }
    invalid = valid
    invalid.Currency = "BHD"
    if err := invalid.Validate(); err == nil { t.Error("accepted non-unity BHD exchange rate") }
    invalid = valid
    invalid.TradeDate = time.Now().UTC().AddDate(0, 0, 3).Format("2006-01-02")
    if err := invalid.Validate(); err == nil { t.Error("accepted a future trade date") }
}

func TestAverageCostBasisIncludingFeesAndPartialSale(t *testing.T) {
    shares, basis, realized, err := ApplyTrade(0, 0, 0, 10, 100, 2, "buy")
    if err != nil || shares != 10 || basis != 102 || realized != 0 {
        t.Fatalf("buy = %v %v %v %v", shares, basis, realized, err)
    }
    shares, basis, realized, err = ApplyTrade(shares, basis, realized, 4, 60, 3, "sell")
    if err != nil || shares != 6 || math.Abs(basis-61.2) > 0.001 || math.Abs(realized-16.2) > 0.001 {
        t.Fatalf("partial sell = %v %v %v %v", shares, basis, realized, err)
    }
    if _, _, _, err = ApplyTrade(shares, basis, realized, 7, 100, 0, "sell"); err == nil {
        t.Fatal("expected oversell rejection")
    }
    shares, basis, realized, err = ApplyTrade(shares, basis, realized, 6, 72, 1, "sell")
    if err != nil || shares != 0 || basis != 0 || math.Abs(realized-26) > 0.001 {
        t.Fatalf("full close = %v %v %v %v", shares, basis, realized, err)
    }
}
