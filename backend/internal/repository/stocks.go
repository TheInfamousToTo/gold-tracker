package repository

import (
    "context"
    "errors"
    "fmt"
    "math"
    "time"

    "github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
    "github.com/jackc/pgx/v5"
)

var ErrInvalidStockTrade = errors.New("invalid stock trade")

func stockInputError(message string) error {
    return fmt.Errorf("%w: %s", ErrInvalidStockTrade, message)
}

func scanBroker(row rowScanner) (model.StockBroker, error) {
    var b model.StockBroker
    err := row.Scan(&b.ID, &b.Name, &b.BuyPercent, &b.SellPercent,
        &b.FixedFee, &b.MinimumFee, &b.TaxPercent)
    return b, err
}

const brokerColumns = "id, name, buy_percent, sell_percent, fixed_fee, minimum_fee, tax_percent"

func (r *PostgresRepository) ListStockBrokers(ctx context.Context) ([]model.StockBroker, error) {
    rows, err := r.Pool.Query(ctx, "SELECT "+brokerColumns+" FROM stock_brokers ORDER BY name")
    if err != nil { return nil, err }
    defer rows.Close()
    brokers := make([]model.StockBroker, 0)
    for rows.Next() {
        broker, err := scanBroker(rows)
        if err != nil { return nil, err }
        brokers = append(brokers, broker)
    }
    return brokers, rows.Err()
}

func (r *PostgresRepository) SaveStockBroker(ctx context.Context, b model.StockBroker) (model.StockBroker, error) {
    if err := b.Validate(); err != nil { return model.StockBroker{}, err }
    if b.ID == 0 {
        return scanBroker(r.Pool.QueryRow(ctx,
            `INSERT INTO stock_brokers (name, buy_percent, sell_percent, fixed_fee, minimum_fee, tax_percent)
             VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+brokerColumns,
            b.Name, b.BuyPercent, b.SellPercent, b.FixedFee, b.MinimumFee, b.TaxPercent))
    }
    return scanBroker(r.Pool.QueryRow(ctx,
        `UPDATE stock_brokers SET name=$2, buy_percent=$3, sell_percent=$4,
          fixed_fee=$5, minimum_fee=$6, tax_percent=$7 WHERE id=$1
          RETURNING `+brokerColumns,
        b.ID, b.Name, b.BuyPercent, b.SellPercent, b.FixedFee, b.MinimumFee, b.TaxPercent))
}

func (r *PostgresRepository) ListStockPositions(ctx context.Context) ([]model.StockPosition, error) {
    rows, err := r.Pool.Query(ctx, `
        SELECT symbol, asset_name, currency, shares, cost_basis_bhd, realized_pnl_bhd,
               latest_price, latest_fx_to_bhd, latest_price_date
        FROM stock_positions ORDER BY (shares > 0) DESC, symbol`)
    if err != nil { return nil, err }
    defer rows.Close()
    positions := make([]model.StockPosition, 0)
    for rows.Next() {
        var p model.StockPosition
        var date *time.Time
        if err := rows.Scan(&p.Symbol, &p.AssetName, &p.Currency, &p.Shares,
            &p.CostBasisBHD, &p.RealizedPNLBHD, &p.LatestPrice,
            &p.LatestFXToBHD, &date); err != nil {
            return nil, err
        }
        if date != nil {
            formatted := date.Format("2006-01-02")
            p.LatestPriceDate = &formatted
        }
        if p.Shares > 0 {
            p.AvgCostBHD = p.CostBasisBHD / p.Shares
        }
        if p.LatestPrice != nil && p.LatestFXToBHD != nil {
            value := math.Round(p.Shares*(*p.LatestPrice)*(*p.LatestFXToBHD)*1000) / 1000
            gain := math.Round((value-p.CostBasisBHD)*1000) / 1000
            p.MarketValueBHD, p.UnrealizedPNLBHD = &value, &gain
        }
        positions = append(positions, p)
    }
    return positions, rows.Err()
}

func (r *PostgresRepository) SaveStockQuote(ctx context.Context, symbol string, q model.StockQuoteInput) error {
    if err := q.Validate(); err != nil { return err }
    command, err := r.Pool.Exec(ctx, `
        UPDATE stock_positions SET latest_price=$2, latest_fx_to_bhd=$3,
        latest_price_date=$4 WHERE symbol=$1`, symbol, q.Price, q.FXToBHD, q.PriceDate)
    if err != nil { return err }
    if command.RowsAffected() == 0 { return pgx.ErrNoRows }
    return nil
}

func (r *PostgresRepository) ListStockTrades(ctx context.Context) ([]model.StockTrade, error) {
    rows, err := r.Pool.Query(ctx, `
        SELECT id, symbol, side, trade_date, shares, unit_price, currency, fx_to_bhd,
               broker_id, broker_name, notional_bhd, fees_bhd, total_bhd,
               realized_pnl_bhd, COALESCE(notes, '')
        FROM stock_trades ORDER BY trade_date DESC, id DESC LIMIT 250`)
    if err != nil { return nil, err }
    defer rows.Close()
    trades := make([]model.StockTrade, 0)
    for rows.Next() {
        var t model.StockTrade
        var date time.Time
        if err := rows.Scan(&t.ID, &t.Symbol, &t.Side, &date, &t.Shares,
            &t.UnitPrice, &t.Currency, &t.FXToBHD, &t.BrokerID,
            &t.BrokerName, &t.NotionalBHD, &t.FeesBHD, &t.TotalBHD,
            &t.RealizedPNLBHD, &t.Notes); err != nil { return nil, err }
        t.TradeDate = date.Format("2006-01-02")
        trades = append(trades, t)
    }
    return trades, rows.Err()
}

// RecordStockTrade locks a symbol's position for the full operation, so
// parallel sell requests cannot oversell. Historical fee totals are stored
// on the trade: editing a broker schedule never rewrites past P/L.
func (r *PostgresRepository) RecordStockTrade(ctx context.Context, input model.StockTradeInput) (model.StockTrade, error) {
    if err := input.Validate(); err != nil { return model.StockTrade{}, stockInputError(err.Error()) }
    tx, err := r.Pool.Begin(ctx)
    if err != nil { return model.StockTrade{}, err }
    defer tx.Rollback(ctx)

    var broker model.StockBroker
    var brokerName *string
    if input.BrokerID != nil {
        broker, err = scanBroker(tx.QueryRow(ctx, "SELECT "+brokerColumns+" FROM stock_brokers WHERE id=$1", *input.BrokerID))
        if errors.Is(err, pgx.ErrNoRows) {
            return model.StockTrade{}, stockInputError("broker not found")
        }
        if err != nil { return model.StockTrade{}, err }
        brokerName = &broker.Name
    }

    // Only a buy can establish a new position. A sell must refer to
    // shares already held.
    if input.Side == "buy" {
        _, err = tx.Exec(ctx, `INSERT INTO stock_positions (symbol, asset_name, currency)
            VALUES ($1, $2, $3) ON CONFLICT (symbol) DO NOTHING`,
            input.Symbol, input.AssetName, input.Currency)
        if err != nil { return model.StockTrade{}, err }
    }

    var heldShares, basis, realized float64
    var currency string
    err = tx.QueryRow(ctx, `SELECT shares, cost_basis_bhd, realized_pnl_bhd, currency
        FROM stock_positions WHERE symbol=$1 FOR UPDATE`, input.Symbol).
        Scan(&heldShares, &basis, &realized, &currency)
    if errors.Is(err, pgx.ErrNoRows) {
        return model.StockTrade{}, stockInputError("no shares held for this symbol")
    }
    if err != nil { return model.StockTrade{}, err }
    if currency != input.Currency {
        return model.StockTrade{}, stockInputError("symbol currency differs from existing position")
    }
    // A moving-average ledger must be entered in trade-date order.
    // Allow multiple trades on the same date in their recorded order.
    var lastDate time.Time
    dateErr := tx.QueryRow(ctx, `SELECT trade_date FROM stock_trades
        WHERE symbol=$1 ORDER BY trade_date DESC, id DESC LIMIT 1`, input.Symbol).Scan(&lastDate)
    if dateErr != nil && !errors.Is(dateErr, pgx.ErrNoRows) {
        return model.StockTrade{}, dateErr
    }
    parsedDate, _ := time.Parse("2006-01-02", input.TradeDate)
    if dateErr == nil && parsedDate.Before(lastDate) {
        return model.StockTrade{}, stockInputError("enter trades in chronological order for each symbol")
    }
    notionalQuote := input.Shares * input.UnitPrice
    feeQuote := broker.Fee(input.Side, notionalQuote)
    notional := math.Round(notionalQuote*input.FXToBHD*1000)/1000
    fee := math.Round(feeQuote*input.FXToBHD*1000)/1000
    if math.IsInf(notional, 0) || math.IsNaN(notional) || math.IsInf(fee, 0) || math.IsNaN(fee) || notional > 1e12 || fee > 1e12 {
        return model.StockTrade{}, stockInputError("trade value too large")
    }
    newShares, newBasis, newRealized, err := model.ApplyTrade(
        heldShares, basis, realized, input.Shares, notional, fee, input.Side)
    if err != nil { return model.StockTrade{}, stockInputError(err.Error()) }

    _, err = tx.Exec(ctx, `UPDATE stock_positions
        SET shares=$2, cost_basis_bhd=$3, realized_pnl_bhd=$4, asset_name=$5
        WHERE symbol=$1`, input.Symbol, newShares, newBasis, newRealized, input.AssetName)
    if err != nil { return model.StockTrade{}, err }

    total := notional + fee
    realizedOnTrade := 0.0
    if input.Side == "sell" {
        total = notional - fee
        realizedOnTrade = math.Round((newRealized-realized)*1000) / 1000
    }
    var id int64
    err = tx.QueryRow(ctx, `INSERT INTO stock_trades
        (symbol, broker_id, broker_name, side, trade_date, shares, unit_price,
         currency, fx_to_bhd, notional_bhd, fees_bhd, total_bhd, realized_pnl_bhd, notes)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id`,
        input.Symbol, input.BrokerID, brokerName, input.Side, input.TradeDate,
        input.Shares, input.UnitPrice, input.Currency, input.FXToBHD,
        notional, fee, total, realizedOnTrade, input.Notes).Scan(&id)
    if err != nil { return model.StockTrade{}, err }
    if err := tx.Commit(ctx); err != nil { return model.StockTrade{}, err }
    return model.StockTrade{
        ID: id, StockTradeInput: input, BrokerName: brokerName,
        NotionalBHD: notional, FeesBHD: fee, TotalBHD: total,
        RealizedPNLBHD: realizedOnTrade,
    }, nil
}

// DeleteLatestStockTrade undoes the most recent trade of a symbol. Only
// the latest may go: removing an earlier one would change the average
// cost every later sale was booked against. The position is rebuilt by
// replaying the remaining trades with their stored BHD amounts, which is
// exactly how it was built, so no fee is recomputed from a broker
// schedule that may have changed since.
func (r *PostgresRepository) DeleteLatestStockTrade(ctx context.Context, id int64) error {
    tx, err := r.Pool.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback(ctx)

    var symbol string
    if err := tx.QueryRow(ctx, `SELECT symbol FROM stock_trades WHERE id=$1`, id).Scan(&symbol); err != nil {
        return err
    }
    if _, err := tx.Exec(ctx, `SELECT 1 FROM stock_positions WHERE symbol=$1 FOR UPDATE`, symbol); err != nil {
        return err
    }
    var latest int64
    if err := tx.QueryRow(ctx, `SELECT id FROM stock_trades WHERE symbol=$1
        ORDER BY trade_date DESC, id DESC LIMIT 1`, symbol).Scan(&latest); err != nil {
        return err
    }
    if latest != id {
        return stockInputError("only the most recent trade for " + symbol + " can be undone")
    }
    if _, err := tx.Exec(ctx, `DELETE FROM stock_trades WHERE id=$1`, id); err != nil { return err }

    rows, err := tx.Query(ctx, `SELECT side, shares, notional_bhd, fees_bhd FROM stock_trades
        WHERE symbol=$1 ORDER BY trade_date, id`, symbol)
    if err != nil { return err }
    var shares, basis, realized float64
    count := 0
    for rows.Next() {
        var side string
        var qty, notional, fee float64
        if err := rows.Scan(&side, &qty, &notional, &fee); err != nil { rows.Close(); return err }
        if shares, basis, realized, err = model.ApplyTrade(shares, basis, realized, qty, notional, fee, side); err != nil {
            rows.Close(); return err
        }
        count++
    }
    rows.Close()
    if err := rows.Err(); err != nil { return err }

    if count == 0 {
        _, err = tx.Exec(ctx, `DELETE FROM stock_positions WHERE symbol=$1`, symbol)
    } else {
        _, err = tx.Exec(ctx, `UPDATE stock_positions SET shares=$2, cost_basis_bhd=$3,
            realized_pnl_bhd=$4 WHERE symbol=$1`, symbol, shares, basis, realized)
    }
    if err != nil { return err }
    return tx.Commit(ctx)
}
