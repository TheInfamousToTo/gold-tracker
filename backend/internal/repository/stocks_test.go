package repository

import (
    "context"
    "math"
    "testing"

    "github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

func TestStockTradesAndBrokerSnapshot(t *testing.T) {
    repo := repoFor(t)
    ctx := context.Background()
    broker, err := repo.SaveStockBroker(ctx, model.StockBroker{
        Name: "Fee Snapshot Test", BuyPercent: 1, SellPercent: 1,
        MinimumFee: 0.5,
    })
    if err != nil { t.Fatalf("broker: %v", err) }
    buy, err := repo.RecordStockTrade(ctx, model.StockTradeInput{
        Symbol: "GLD", AssetName: "Gold ETF", Side: "buy",
        TradeDate: "2026-11-01", Shares: 10, UnitPrice: 10,
        Currency: "BHD", FXToBHD: 1, BrokerID: &broker.ID,
    })
    if err != nil { t.Fatalf("buy: %v", err) }
    if buy.FeesBHD != 1 || buy.TotalBHD != 101 { t.Fatalf("buy costs: %+v", buy) }

    sold, err := repo.RecordStockTrade(ctx, model.StockTradeInput{
        Symbol: "GLD", AssetName: "Gold ETF", Side: "sell",
        TradeDate: "2026-11-02", Shares: 4, UnitPrice: 15,
        Currency: "BHD", FXToBHD: 1, BrokerID: &broker.ID,
    })
    if err != nil { t.Fatalf("sell: %v", err) }
    // Proceeds 60 - 0.6 fee - 40.4 proportional basis = 19 BHD.
    if math.Abs(sold.RealizedPNLBHD-19) > 0.001 {
        t.Errorf("realized P/L: %v, want 19", sold.RealizedPNLBHD)
    }
    if _, err := repo.RecordStockTrade(ctx, model.StockTradeInput{
        Symbol: "GLD", AssetName: "Gold ETF", Side: "sell",
        TradeDate: "2026-11-03", Shares: 7, UnitPrice: 15,
        Currency: "BHD", FXToBHD: 1,
    }); err == nil {
        t.Fatal("oversold stock position")
    }

    positions, err := repo.ListStockPositions(ctx)
    if err != nil { t.Fatal(err) }
    if len(positions) != 1 || positions[0].Shares != 6 ||
        math.Abs(positions[0].CostBasisBHD-60.6) > 0.001 {
        t.Fatalf("position = %+v", positions)
    }
    if positions[0].MarketValueBHD != nil {
        t.Fatal("unquoted position was valued as zero")
    }

    if err := repo.SaveStockQuote(ctx, "GLD", model.StockQuoteInput{
        Price: 20, FXToBHD: 1, PriceDate: "2026-11-03",
    }); err != nil { t.Fatal(err) }
    positions, err = repo.ListStockPositions(ctx)
    if err != nil { t.Fatal(err) }
    if positions[0].MarketValueBHD == nil || *positions[0].MarketValueBHD != 120 {
        t.Fatalf("market value = %+v, want 120", positions[0].MarketValueBHD)
    }

    _, err = repo.SaveStockBroker(ctx, model.StockBroker{
        ID: broker.ID, Name: broker.Name, BuyPercent: 50, SellPercent: 50,
    })
    if err != nil { t.Fatal(err) }
    trades, err := repo.ListStockTrades(ctx)
    if err != nil { t.Fatal(err) }
    if len(trades) != 2 || trades[1].FeesBHD != 1 || trades[0].FeesBHD != 0.6 {
        t.Fatalf("historic fees were changed: %+v", trades)
    }
}

func TestUndoLatestStockTradeReplaysPosition(t *testing.T) {
    repo := repoFor(t)
    ctx := context.Background()
    trade := func(side, date string, shares, price float64) model.StockTrade {
        t.Helper()
        saved, err := repo.RecordStockTrade(ctx, model.StockTradeInput{
            Symbol: "VOO", AssetName: "S&P 500 ETF", Side: side, TradeDate: date,
            Shares: shares, UnitPrice: price, Currency: "BHD", FXToBHD: 1,
        })
        if err != nil { t.Fatalf("%s: %v", side, err) }
        return saved
    }
    first := trade("buy", "2026-10-01", 10, 10)
    trade("buy", "2026-10-02", 10, 20)
    sell := trade("sell", "2026-10-03", 5, 30)

    if err := repo.DeleteLatestStockTrade(ctx, first.ID); err == nil {
        t.Fatal("undid a trade that later trades depend on")
    }
    if err := repo.DeleteLatestStockTrade(ctx, sell.ID); err != nil { t.Fatal(err) }
    positions, err := repo.ListStockPositions(ctx)
    if err != nil { t.Fatal(err) }
    if positions[0].Shares != 20 || positions[0].CostBasisBHD != 300 || positions[0].RealizedPNLBHD != 0 {
        t.Fatalf("after undoing the sale: %+v", positions[0])
    }

    trades, _ := repo.ListStockTrades(ctx)
    for _, tr := range trades {
        if err := repo.DeleteLatestStockTrade(ctx, tr.ID); err != nil { t.Fatal(err) }
    }
    if positions, _ = repo.ListStockPositions(ctx); len(positions) != 0 {
        t.Fatalf("position left after undoing every trade: %+v", positions)
    }
}
