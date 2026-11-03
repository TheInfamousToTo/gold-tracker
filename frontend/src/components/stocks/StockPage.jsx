import { useCallback, useEffect, useMemo, useState } from 'react';
import { apiRequest } from '../../api/client.js';
import { fmt, fmtDate } from '../../lib/format.js';
import { estimateTrade, stockTotals } from '../../lib/stocks.js';
import { Card } from '../ui/Card.jsx';
import { Button } from '../ui/Button.jsx';
import { Field, inputClass } from '../ui/Field.jsx';

// Local date, not toISOString(): in Bahrain (UTC+3) that reads yesterday until 03:00.
const today = () => new Date().toLocaleDateString('en-CA');
const blankTrade = () => ({
  side: 'buy', trade_date: today(), symbol: '', asset_name: '',
  shares: '', unit_price: '', currency: 'BHD', fx_to_bhd: '1',
  broker_id: '', notes: '',
});
const blankBroker = () => ({
  id: null, name: '', buy_percent: '0', sell_percent: '0',
  fixed_fee: '0', minimum_fee: '0', tax_percent: '0',
});

function money(value) {
  return value == null ? '—' : fmt(value, 3);
}
function Tone({ value }) {
  if (value == null) return <span className="text-muted">—</span>;
  return <span className={Number(value) >= 0 ? 'text-ok-bright' : 'text-bad-bright'}>
    {Number(value) >= 0 ? '+' : ''}{money(value)}
  </span>;
}
function SummaryCell({ title, value, detail }) {
  return (
    <div className="rounded-lg border border-line bg-ink-raised px-4 py-4">
      <p className="stamp">{title}</p>
      <p className="mt-2 font-mono text-xl font-semibold text-chalk">{value}</p>
      <p className="mt-1 text-xs text-muted">{detail}</p>
    </div>
  );
}

export function StockPage() {
  const [positions, setPositions] = useState([]);
  const [brokers, setBrokers] = useState([]);
  const [trades, setTrades] = useState([]);
  const [trade, setTrade] = useState(blankTrade);
  const [brokerForm, setBrokerForm] = useState(blankBroker);
  const [quoteSymbol, setQuoteSymbol] = useState('');
  const [quote, setQuote] = useState({ price: '', fx_to_bhd: '1', price_date: today() });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState(null);
  const [error, setError] = useState(null);

  const refresh = useCallback(async () => {
    try {
      const [p, b, t] = await Promise.all([
        apiRequest('/api/stocks/positions'),
        apiRequest('/api/stocks/brokers'),
        apiRequest('/api/stocks/trades'),
      ]);
      setPositions(p || []);
      setBrokers(b || []);
      setTrades(t || []);
      setError(null);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => { refresh(); }, [refresh]);

  const totals = useMemo(() => stockTotals(positions), [positions]);
  const activePositions = positions.filter((p) => Number(p.shares) > 0);
  const selectedBroker = brokers.find((b) => String(b.id) === String(trade.broker_id));
  const preview = estimateTrade(trade, selectedBroker);
  const setTradeField = (field) => (e) => setTrade((f) => ({ ...f, [field]: e.target.value }));
  const setBrokerField = (field) => (e) => setBrokerForm((b) => ({ ...b, [field]: e.target.value }));
  const setQuoteField = (field) => (e) => setQuote((q) => ({ ...q, [field]: e.target.value }));

  const selectSymbol = (e) => {
    const symbol = e.target.value.toUpperCase();
    const p = positions.find((s) => s.symbol === symbol);
    setTrade((t) => ({
      ...t, symbol, asset_name: p?.asset_name || t.asset_name,
      currency: p?.currency || t.currency,
      fx_to_bhd: p?.currency === 'BHD' ? '1' : (p ? '' : t.fx_to_bhd),
    }));
  };
  const changeCurrency = (e) => {
    const currency = e.target.value.toUpperCase();
    setTrade((t) => ({ ...t, currency, fx_to_bhd: currency === 'BHD' ? '1' : '' }));
  };
  const submitTrade = async (e) => {
    e.preventDefault();
    setSaving(true); setNotice(null); setError(null);
    try {
      const payload = {
        ...trade,
        symbol: trade.symbol.trim().toUpperCase(),
        shares: Number(trade.shares),
        unit_price: Number(trade.unit_price),
        fx_to_bhd: trade.currency === 'BHD' ? 1 : Number(trade.fx_to_bhd),
        broker_id: trade.broker_id ? Number(trade.broker_id) : null,
      };
      const saved = await apiRequest('/api/stocks/trades', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      setNotice((saved.side === 'sell' ? 'Sale' : 'Purchase') +
        ' recorded: ' + saved.symbol + '. Fees: ' + money(saved.fees_bhd) + ' BHD');
      setTrade(blankTrade());
      await refresh();
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  };

  const submitBroker = async (e) => {
    e.preventDefault();
    setSaving(true); setNotice(null); setError(null);
    try {
      const { id, ...values } = brokerForm;
      const payload = Object.fromEntries(Object.entries(values).map(([k, v]) =>
        k === 'name' ? [k, v.trim()] : [k, Number(v)]
      ));
      await apiRequest(id ? '/api/stocks/brokers/' + id : '/api/stocks/brokers', {
        method: id ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      setNotice('Broker fee schedule saved');
      setBrokerForm(blankBroker());
      await refresh();
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  };

  // The trade list is newest first, so the first row per symbol is the
  // only one the API will undo.
  const undoable = new Set();
  const seen = new Set();
  for (const t of trades) { if (!seen.has(t.symbol)) { seen.add(t.symbol); undoable.add(t.id); } }
  const undoTrade = async (t) => {
    if (!confirm(`Undo ${t.side} of ${fmt(t.shares, 4)} ${t.symbol} on ${fmtDate(t.trade_date)}? This can't be undone.`)) return;
    setSaving(true); setNotice(null); setError(null);
    try {
      await apiRequest('/api/stocks/trades/' + t.id, { method: 'DELETE' });
      setNotice('Trade undone: ' + t.symbol);
      await refresh();
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  };

  const startQuote = (p) => {
    setQuoteSymbol(p.symbol);
    setQuote({
      price: p.latest_price == null ? '' : String(p.latest_price),
      fx_to_bhd: p.currency === 'BHD' ? '1' : (p.latest_fx_to_bhd == null ? '' : String(p.latest_fx_to_bhd)),
      price_date: today(),
    });
    setNotice(null);
  };
  const submitQuote = async (e) => {
    e.preventDefault();
    setSaving(true); setNotice(null); setError(null);
    try {
      await apiRequest('/api/stocks/quotes/' + encodeURIComponent(quoteSymbol), {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          price: Number(quote.price), fx_to_bhd: Number(quote.fx_to_bhd),
          price_date: quote.price_date,
        }),
      });
      setNotice('Manual quote updated for ' + quoteSymbol);
      setQuoteSymbol('');
      await refresh();
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-display text-xl font-semibold text-chalk">Stocks & ETFs</h1>
        <p className="mt-1 text-sm text-muted">Track shares separately from physical gold and silver. All totals are converted to BHD using your entered FX rates.</p>
      </div>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <SummaryCell title="Open cost basis" value={money(totals.cost) + ' BHD'} detail="Includes purchase commissions" />
        <SummaryCell title="Quoted market value" value={money(totals.value) + ' BHD'} detail={totals.unpriced ? totals.unpriced + ' open positions have no quote — partial total' : 'Using manually entered quotes'} />
        <SummaryCell title="Unrealized P/L" value={<Tone value={totals.unpriced === activePositions.length && totals.unpriced ? null : totals.unrealized} />} detail="Before estimated future sale fees" />
        <SummaryCell title="Realized P/L" value={<Tone value={totals.realized} />} detail="Net of fees on completed sales" />
      </div>
      {loading && <p className="text-sm text-muted">Loading stock positions…</p>}
      {error && <p role="alert" className="rounded-chip border border-bad/30 bg-bad/10 p-3 text-sm text-bad-bright">{error}</p>}
      {notice && <p role="status" className="rounded-chip border border-ok/30 bg-ok/10 p-3 text-sm text-ok-bright">{notice}</p>}

      <Card title="Record stock trade">
        <form onSubmit={submitTrade} className="space-y-4">
          <div className="flex gap-2" role="group" aria-label="Trade side">
            {['buy', 'sell'].map((side) => (
              <button type="button" key={side}
                aria-pressed={trade.side === side}
                onClick={() => setTrade((t) => ({ ...t, side }))}
                className={'rounded-chip border px-5 py-2 text-sm font-semibold uppercase ' +
                  (trade.side === side ? 'border-chalk bg-chalk text-ink' : 'border-line text-muted hover:text-chalk')}>
                {side}
              </button>
            ))}
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <Field label="Ticker" htmlFor="stock_symbol">
              <input required id="stock_symbol" type="text" maxLength="16" list="stock-symbols"
                placeholder="e.g. GLD or AAPL" className={inputClass} value={trade.symbol} onChange={selectSymbol} />
              <datalist id="stock-symbols">{positions.map((p) => <option key={p.symbol} value={p.symbol}>{p.asset_name}</option>)}</datalist>
            </Field>
            <Field label="Stock / ETF name" htmlFor="stock_name">
              <input id="stock_name" required maxLength="120" value={trade.asset_name}
                onChange={setTradeField('asset_name')} placeholder="Asset name" className={inputClass} />
            </Field>
            <Field label="Trade date" htmlFor="stock_date">
              <input id="stock_date" required type="date" value={trade.trade_date} onChange={setTradeField('trade_date')} className={inputClass} />
            </Field>
            <Field label="Shares" htmlFor="stock_shares">
              <input id="stock_shares" required type="number" step="any" min="0.00000001"
                value={trade.shares} onChange={setTradeField('shares')} className={inputClass} />
            </Field>
            <Field label="Price per share" htmlFor="stock_unit">
              <input id="stock_unit" required type="number" step="any" min="0.000001"
                value={trade.unit_price} onChange={setTradeField('unit_price')} className={inputClass} />
            </Field>
            <Field label="Quote currency" htmlFor="stock_currency">
              <select id="stock_currency" className={inputClass} value={trade.currency} onChange={changeCurrency}>
                {['BHD', 'USD', 'EUR', 'GBP', 'SAR', 'AED', 'KWD'].map((cur) => <option key={cur} value={cur}>{cur}</option>)}
              </select>
            </Field>
            {trade.currency !== 'BHD' && (
              <Field label="1 quote currency = BHD" htmlFor="stock_fx" hint="Enter actual FX rate for this trade; not fetched automatically">
                <input id="stock_fx" required type="number" min="0.00000001" step="any"
                  value={trade.fx_to_bhd} onChange={setTradeField('fx_to_bhd')} className={inputClass} />
              </Field>
            )}
            <Field label="Broker fee schedule" htmlFor="stock_broker" hint="Choose a saved broker to calculate fees">
              <select id="stock_broker" value={trade.broker_id} onChange={setTradeField('broker_id')} className={inputClass}>
                <option value="">No brokerage fees</option>
                {brokers.map((b) => <option value={b.id} key={b.id}>{b.name}</option>)}
              </select>
            </Field>
            <Field label="Notes" htmlFor="stock_notes">
              <input id="stock_notes" maxLength="2000" value={trade.notes} onChange={setTradeField('notes')}
                className={inputClass} placeholder="Order reference (optional)" />
            </Field>
          </div>
          <div className="rounded-chip border border-line bg-ink-sunken p-4 text-sm">
            <p className="stamp">Estimated trade settlement · BHD</p>
            <div className="mt-2 flex flex-wrap items-center gap-5 font-mono text-chalk">
              <span>Gross: {money(preview?.gross)}</span>
              <span>Broker fees: {money(preview?.fee)}</span>
              <strong>{trade.side === 'sell' ? 'Net proceeds' : 'Total cost'}: {money(preview?.total)}</strong>
            </div>
            <p className="mt-2 text-xs text-muted">Charges follow your saved broker schedule (commission + fixed/minimum + tax on fees). No broker selected means zero fees. Verify the broker's actual tariff and execution statement.</p>
          </div>
          <Button type="submit" size="lg" loading={saving} loadingLabel="Recording">
            Record {trade.side === 'sell' ? 'sale' : 'purchase'}
          </Button>
        </form>
      </Card>

      <Card title="Stock holdings" padded={false}>
        {activePositions.length === 0 ? (
          <p className="p-5 text-sm text-muted">No open stock positions. Record a buy above to start.</p>
        ) : (
          <div className="overflow-x-auto"><table className="w-full text-sm">
            <thead><tr className="stamp text-left">
              {['Stock', 'Shares', 'Avg. cost BHD', 'Last manual quote', 'Market value BHD', 'Unrealized BHD', 'Realized BHD', ''].map((label) =>
                <th key={label} className="whitespace-nowrap px-4 py-3">{label}</th>)}
            </tr></thead>
            <tbody>{activePositions.map((p) => (
              <tr key={p.symbol} className="border-t border-line">
                <td className="px-4 py-3 text-chalk"><strong>{p.symbol}</strong><p className="text-xs text-muted">{p.asset_name}</p></td>
                <td className="px-4 py-3 font-mono">{fmt(p.shares, 4)}</td>
                <td className="px-4 py-3 font-mono">{money(p.avg_cost_bhd)}</td>
                <td className="px-4 py-3 font-mono">{p.latest_price == null ? 'Unpriced' : money(p.latest_price) + ' ' + p.currency}<p className="text-xs text-muted">{p.latest_price_date ? fmtDate(p.latest_price_date) + ' · Manual' : 'No quote yet'}</p></td>
                <td className="px-4 py-3 font-mono">{money(p.market_value_bhd)}</td>
                <td className="px-4 py-3 font-mono"><Tone value={p.unrealized_pnl_bhd} /></td>
                <td className="px-4 py-3 font-mono"><Tone value={p.realized_pnl_bhd} /></td>
                <td className="px-4 py-3"><button type="button" onClick={() => startQuote(p)} className="whitespace-nowrap text-xs font-semibold text-chalk underline underline-offset-4 hover:text-brand">Update quote</button></td>
              </tr>
            ))}</tbody>
          </table></div>
        )}
        {quoteSymbol && (
          <form onSubmit={submitQuote} className="space-y-3 border-t border-line p-5">
            <p className="font-semibold text-chalk">Manual market quote · {quoteSymbol}</p>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <Field label={'Price (' + positions.find((p) => p.symbol === quoteSymbol)?.currency + ')'} htmlFor="quote_price">
                <input id="quote_price" type="number" required step="any" min="0.000001" value={quote.price}
                  onChange={setQuoteField('price')} className={inputClass} />
              </Field>
              <Field label="FX to BHD" htmlFor="quote_fx">
                <input id="quote_fx" required type="number" step="any" min="0.00000001"
                  value={quote.fx_to_bhd} onChange={setQuoteField('fx_to_bhd')} className={inputClass}
                  disabled={positions.find((p) => p.symbol === quoteSymbol)?.currency === 'BHD'} />
              </Field>
              <Field label="Quote date" htmlFor="quote_date">
                <input id="quote_date" required type="date" value={quote.price_date}
                  onChange={setQuoteField('price_date')} className={inputClass} />
              </Field>
            </div>
            <div className="flex gap-3">
              <Button type="submit" loading={saving}>Save quote</Button>
              <Button type="button" variant="ghost" onClick={() => setQuoteSymbol('')}>Cancel</Button>
            </div>
          </form>
        )}
      </Card>

      <Card title="Broker fee schedules">
        <p className="mb-4 text-sm text-muted">Save a broker once and reuse its charges when you trade. Fixed and minimum amounts are in the quote currency. Existing trades keep their original fee amounts when a profile changes.</p>
        {brokers.length > 0 && (
          <div className="mb-4 flex flex-wrap gap-2">
            {brokers.map((b) => (
              <button type="button" key={b.id} onClick={() => setBrokerForm({
                ...b, id: b.id, buy_percent: String(b.buy_percent), sell_percent: String(b.sell_percent),
                fixed_fee: String(b.fixed_fee), minimum_fee: String(b.minimum_fee), tax_percent: String(b.tax_percent),
              })} className="rounded-chip border border-line px-3 py-2 text-xs text-chalk hover:border-chalk/50">
                Edit {b.name}
              </button>
            ))}
          </div>
        )}
        <form onSubmit={submitBroker} className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <Field label="Broker name" htmlFor="broker_name">
              <input id="broker_name" required maxLength="120" value={brokerForm.name}
                onChange={setBrokerField('name')} className={inputClass} placeholder="My brokerage" />
            </Field>
            {[
              ['buy_percent', 'Buy commission (%)'],
              ['sell_percent', 'Sell commission (%)'],
              ['fixed_fee', 'Fixed fee per trade'],
              ['minimum_fee', 'Minimum commission'],
              ['tax_percent', 'Tax on commission (%)'],
            ].map(([field, label]) => (
              <Field key={field} label={label} htmlFor={'broker_' + field}>
                <input id={'broker_' + field} type="number" required step="any" min="0"
                  max={field.includes('percent') ? 100 : undefined}
                  value={brokerForm[field]} onChange={setBrokerField(field)} className={inputClass} />
              </Field>
            ))}
          </div>
          <div className="flex gap-3">
            <Button type="submit" loading={saving}>{brokerForm.id ? 'Update broker' : 'Save broker'}</Button>
            {brokerForm.id && <Button type="button" variant="ghost" onClick={() => setBrokerForm(blankBroker())}>New broker</Button>}
          </div>
        </form>
      </Card>

      <Card title="Trade history" padded={false}>
        {trades.length === 0 ? <p className="p-5 text-sm text-muted">No trades recorded yet.</p> : (
          <div className="overflow-x-auto"><table className="w-full text-sm">
            <thead><tr className="stamp text-left">
              {['Date', 'Ticker', 'Side', 'Shares', 'Price', 'Broker', 'Fees BHD', 'Settlement BHD', 'Realized P/L BHD', ''].map((label) =>
                <th key={label} className="whitespace-nowrap px-4 py-3">{label}</th>)}
            </tr></thead>
            <tbody>{trades.map((t) => (
              <tr key={t.id} className="border-t border-line">
                <td className="whitespace-nowrap px-4 py-3">{fmtDate(t.trade_date)}</td>
                <td className="px-4 py-3 font-semibold text-chalk">{t.symbol}</td>
                <td className="px-4 py-3 uppercase">{t.side}</td>
                <td className="px-4 py-3 font-mono">{fmt(t.shares, 4)}</td>
                <td className="whitespace-nowrap px-4 py-3 font-mono">{money(t.unit_price)} {t.currency}</td>
                <td className="px-4 py-3">{t.broker_name || 'None'}</td>
                <td className="px-4 py-3 font-mono">{money(t.fees_bhd)}</td>
                <td className="px-4 py-3 font-mono">{money(t.total_bhd)}</td>
                <td className="px-4 py-3 font-mono">{t.side === 'sell' ? <Tone value={t.realized_pnl_bhd} /> : '—'}</td>
                <td className="px-4 py-3">{undoable.has(t.id) &&
                  <Button variant="danger" size="sm" disabled={saving} onClick={() => undoTrade(t)}>Undo</Button>}</td>
              </tr>
            ))}</tbody>
          </table></div>
        )}
      </Card>
    </div>
  );
}
