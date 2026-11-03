// Mirrors backend fee preview. Backend remains authoritative for saved totals.
export function brokerFee(broker, side, notional) {
  if (!broker || !Number.isFinite(notional) || notional < 0) return 0;
  const percent = Number(side === 'sell' ? broker.sell_percent : broker.buy_percent) || 0;
  const fixed = Number(broker.fixed_fee) || 0;
  const minimum = Number(broker.minimum_fee) || 0;
  const tax = Number(broker.tax_percent) || 0;
  return Math.max(notional * percent / 100 + fixed, minimum) * (1 + tax / 100);
}

export function estimateTrade(form, broker) {
  const shares = Number(form.shares);
  const price = Number(form.unit_price);
  const rate = form.currency === 'BHD' ? 1 : Number(form.fx_to_bhd);
  if (![shares, price, rate].every((v) => Number.isFinite(v) && v > 0)) return null;
  const gross = Math.round(shares * price * rate * 1000) / 1000;
  const fee = Math.round(brokerFee(broker, form.side, shares * price) * rate * 1000) / 1000;
  return {
    gross,
    fee,
    total: form.side === 'sell' ? gross - fee : gross + fee,
  };
}

export function stockTotals(positions) {
  const active = positions.filter((p) => Number(p.shares) > 0);
  return {
    cost: active.reduce((sum, p) => sum + Number(p.cost_basis_bhd || 0), 0),
    value: active.reduce((sum, p) => sum + Number(p.market_value_bhd || 0), 0),
    unrealized: active.reduce((sum, p) => sum + Number(p.unrealized_pnl_bhd || 0), 0),
    realized: positions.reduce((sum, p) => sum + Number(p.realized_pnl_bhd || 0), 0),
    unpriced: active.filter((p) => p.market_value_bhd == null).length,
  };
}
