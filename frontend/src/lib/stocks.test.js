import { describe, it, expect } from 'vitest';
import { brokerFee, estimateTrade, stockTotals } from './stocks.js';

describe('brokerage fees', () => {
  const broker = { buy_percent: 0.5, sell_percent: 1, fixed_fee: 0.5, minimum_fee: 2, tax_percent: 10 };
  it('applies a minimum and tax to buy-side commissions', () => {
    expect(brokerFee(broker, 'buy', 100)).toBeCloseTo(2.2);
  });
  it('applies the correct sell-side rate', () => {
    expect(brokerFee(broker, 'sell', 1000)).toBeCloseTo(11.55);
  });
  it('converts amounts to BHD and deducts selling fees', () => {
    const preview = estimateTrade({ side: 'sell', shares: '2', unit_price: '100', currency: 'USD', fx_to_bhd: '0.377' }, broker);
    expect(preview.gross).toBe(75.4);
    expect(preview.fee).toBe(0.944);
    expect(preview.total).toBeCloseTo(74.456);
  });
  it('requires a positive FX rate for foreign-currency shares', () => {
    expect(estimateTrade({ side: 'buy', shares: 1, unit_price: 4, currency: 'USD', fx_to_bhd: 0 }, null)).toBeNull();
  });
  it('does not treat missing quotes as zero-valued holdings', () => {
    expect(stockTotals([
      { shares: 2, cost_basis_bhd: 20, market_value_bhd: 25, unrealized_pnl_bhd: 5, realized_pnl_bhd: 2 },
      { shares: 3, cost_basis_bhd: 30, market_value_bhd: null, realized_pnl_bhd: 0 },
    ])).toEqual({ cost: 50, value: 25, unrealized: 5, realized: 2, unpriced: 1 });
  });
});
