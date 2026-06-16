import React, { useState, useEffect, useCallback, useMemo } from 'react';

const KARAT_OPTIONS = [
  { value: '24', label: '24K · 99.9%' },
  { value: '22', label: '22K · 91.6%' },
  { value: '21', label: '21K · 87.5%' },
  { value: '18', label: '18K · 75.0%' },
];

const EMPTY_ITEM_FORM = {
  purchase_date: new Date().toISOString().split('T')[0],
  item_name: '',
  metal_type: 'gold',
  purity_karat: '21',
  weight_grams: '',
  price_paid_total: '',
  vendor: '',
  notes: '',
};

const EMPTY_PRICE_FORM = {
  price_date: new Date().toISOString().split('T')[0],
  price_per_gram_24k: '',
};

const TABS = [
  { id: 'holdings', label: 'Holdings' },
  { id: 'add-item', label: 'Add purchase' },
  { id: 'prices', label: 'Prices & signals' },
];

function fmt(value, decimals = 3) {
  const n = Number(value);
  return Number.isFinite(n) ? n.toFixed(decimals) : (0).toFixed(decimals);
}

function fmtDate(value) {
  if (!value) return '—';
  return new Date(value).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

async function apiRequest(url, options) {
  const res = await fetch(url, options);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(body.error || `Request failed (${res.status})`);
  }
  return body;
}

export default function App() {
  const [activeTab, setActiveTab] = useState('holdings');
  const [portfolio, setPortfolio] = useState({ items: [], totals: {}, has_price_data: false });
  const [prices, setPrices] = useState([]);
  const [signals, setSignals] = useState([]);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [toast, setToast] = useState(null);

  const [itemForm, setItemForm] = useState(EMPTY_ITEM_FORM);
  const [priceForm, setPriceForm] = useState(EMPTY_PRICE_FORM);
  const [editingId, setEditingId] = useState(null);
  const [submitting, setSubmitting] = useState(false);

  const showToast = useCallback((message, kind = 'success') => {
    setToast({ message, kind });
    setTimeout(() => setToast(null), 3000);
  }, []);

  const refreshData = useCallback(async () => {
    setError(null);
    try {
      const [portfolioData, pricesData, signalsData] = await Promise.all([
        apiRequest('/api/portfolio'),
        apiRequest('/api/prices'),
        apiRequest('/api/signals'),
      ]);
      setPortfolio(portfolioData);
      setPrices(pricesData);
      setSignals(signalsData);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refreshData();
  }, [refreshData]);

  const handleItemSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      if (editingId) {
        await apiRequest(`/api/items/${editingId}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(itemForm),
        });
        showToast('Purchase updated');
      } else {
        await apiRequest('/api/items', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(itemForm),
        });
        showToast('Purchase added');
      }
      setItemForm(EMPTY_ITEM_FORM);
      setEditingId(null);
      await refreshData();
      setActiveTab('holdings');
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      setSubmitting(false);
    }
  };

  const handleEdit = (item) => {
    setItemForm({
      purchase_date: item.purchase_date.split('T')[0],
      item_name: item.item_name,
      metal_type: item.metal_type || 'gold',
      purity_karat: String(item.purity_karat),
      weight_grams: String(item.weight_grams),
      price_paid_total: String(item.price_paid_total),
      vendor: item.vendor || '',
      notes: item.notes || '',
    });
    setEditingId(item.id);
    setActiveTab('add-item');
  };

  const cancelEdit = () => {
    setItemForm(EMPTY_ITEM_FORM);
    setEditingId(null);
  };

  const handlePriceSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await apiRequest('/api/prices', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(priceForm),
      });
      setPriceForm(EMPTY_PRICE_FORM);
      showToast('Price saved');
      await refreshData();
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      setSubmitting(false);
    }
  };

  const deleteItem = async (id, name) => {
    if (!confirm(`Remove "${name}" from your holdings? This can't be undone.`)) return;
    try {
      await apiRequest(`/api/items/${id}`, { method: 'DELETE' });
      showToast('Purchase removed');
      await refreshData();
    } catch (err) {
      showToast(err.message, 'error');
    }
  };

  const totals = portfolio.totals || {};
  const isGain = (totals.total_gain_loss || 0) >= 0;

  const priceChange = useMemo(() => {
    if (prices.length < 2) return null;
    const latest = Number(prices[0].price_per_gram_24k);
    const prev = prices[1] ? Number(prices[1].price_per_gram_24k) : null;
    if (prev === null) return null;
    const diff = latest - prev;
    const pct = prev !== 0 ? (diff / prev) * 100 : 0;
    return { diff, pct, latest };
  }, [prices]);

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100">
      <header className="sticky top-0 z-10 border-b border-gray-800 bg-gray-950/80 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-2.5">
            <span className="text-2xl">🪙</span>
            <span className="text-lg font-semibold tracking-tight">Gold Tracker</span>
          </div>
          <nav className="flex gap-1">
            {TABS.map((tab) => (
              <button
                key={tab.id}
                onClick={() => {
                  if (tab.id !== 'add-item') cancelEdit();
                  setActiveTab(tab.id);
                }}
                className={`rounded-md px-3 py-2 text-sm font-medium transition ${
                  activeTab === tab.id
                    ? 'bg-amber-500/10 text-amber-400'
                    : 'text-gray-400 hover:text-gray-200'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </nav>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
        {error && (
          <div className="mb-6 rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-300">
            Couldn't reach the server: {error}.{' '}
            <button onClick={refreshData} className="underline hover:text-red-200">
              Try again
            </button>
          </div>
        )}

        {activeTab === 'holdings' && (
          <Holdings
            portfolio={portfolio}
            totals={totals}
            isGain={isGain}
            loading={loading}
            refreshData={refreshData}
            onEdit={handleEdit}
            onDelete={deleteItem}
          />
        )}

        {activeTab === 'add-item' && (
          <ItemForm
            itemForm={itemForm}
            setItemForm={setItemForm}
            onSubmit={handleItemSubmit}
            editingId={editingId}
            onCancel={cancelEdit}
            submitting={submitting}
          />
        )}

        {activeTab === 'prices' && (
          <PricesAndSignals
            priceForm={priceForm}
            setPriceForm={setPriceForm}
            onSubmit={handlePriceSubmit}
            prices={prices}
            signals={signals}
            priceChange={priceChange}
            submitting={submitting}
          />
        )}
      </main>

      {toast && (
        <div
          className={`fixed bottom-6 right-6 rounded-lg border px-4 py-3 text-sm shadow-lg ${
            toast.kind === 'error'
              ? 'border-red-500/30 bg-red-950 text-red-200'
              : 'border-emerald-500/30 bg-emerald-950 text-emerald-200'
          }`}
        >
          {toast.message}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------

function StatCard({ label, value, unit, tone = 'default' }) {
  const toneClass =
    tone === 'gain' ? 'text-emerald-400' : tone === 'loss' ? 'text-red-400' : tone === 'accent' ? 'text-amber-400' : 'text-gray-100';

  return (
    <div className="rounded-xl border border-gray-800 bg-gray-900 p-5">
      <p className="text-xs font-medium uppercase tracking-wide text-gray-500">{label}</p>
      <p className={`mt-2 text-2xl font-semibold ${toneClass}`}>
        {value}
        {unit && <span className="ml-1 text-sm font-normal text-gray-500">{unit}</span>}
      </p>
    </div>
  );
}

function Holdings({ portfolio, totals, isGain, loading, refreshData, onEdit, onDelete }) {
  const items = portfolio.items || [];

  return (
    <div className="space-y-6">
      {!portfolio.has_price_data && items.length > 0 && (
        <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-300">
          No spot price recorded yet, so current values can't be calculated. Add one under
          <span className="font-medium"> Prices &amp; signals</span>.
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Total cost" value={fmt(totals.total_paid)} unit="BHD" />
        <StatCard label="Current value" value={fmt(totals.total_value)} unit="BHD" tone="accent" />
        <StatCard
          label="Gain / loss"
          value={`${isGain ? '+' : ''}${fmt(totals.total_gain_loss)}`}
          unit="BHD"
          tone={isGain ? 'gain' : 'loss'}
        />
        <StatCard
          label="Return"
          value={`${isGain ? '+' : ''}${fmt(totals.total_gain_loss_pct, 2)}%`}
          tone={isGain ? 'gain' : 'loss'}
        />
      </div>

      <div className="overflow-hidden rounded-xl border border-gray-800 bg-gray-900">
        <div className="flex items-center justify-between border-b border-gray-800 px-5 py-4">
          <h2 className="text-base font-medium text-gray-200">Your holdings</h2>
          <button onClick={refreshData} className="text-sm text-gray-400 transition hover:text-amber-400">
            {loading ? 'Refreshing…' : 'Refresh'}
          </button>
        </div>

        {items.length === 0 ? (
          <div className="px-5 py-16 text-center">
            <p className="text-sm text-gray-400">
              {loading ? 'Loading your holdings…' : 'No purchases logged yet.'}
            </p>
            {!loading && <p className="mt-1 text-xs text-gray-500">Use "Add purchase" to record your first piece.</p>}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-gray-800 text-xs uppercase tracking-wide text-gray-500">
                <tr>
                  <th className="px-5 py-3">Item</th>
                  <th className="px-5 py-3">Date</th>
                  <th className="px-5 py-3">Purity</th>
                  <th className="px-5 py-3 text-right">Weight</th>
                  <th className="px-5 py-3 text-right">Cost</th>
                  <th className="px-5 py-3 text-right">Value</th>
                  <th className="px-5 py-3 text-right">Gain / loss</th>
                  <th className="px-5 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-800">
                {items.map((item) => {
                  const gain = Number(item.gain_loss) || 0;
                  const hasValue = item.current_value !== null;
                  return (
                    <tr key={item.id} className="transition hover:bg-gray-800/40">
                      <td className="px-5 py-3.5">
                        <div className="font-medium text-gray-100">{item.item_name}</div>
                        {item.vendor && <div className="text-xs text-gray-500">{item.vendor}</div>}
                      </td>
                      <td className="px-5 py-3.5 text-gray-400">{fmtDate(item.purchase_date)}</td>
                      <td className="px-5 py-3.5">
                        <span className="rounded bg-gray-800 px-2 py-0.5 font-mono text-xs text-gray-300">
                          {item.purity_karat}K
                        </span>
                      </td>
                      <td className="px-5 py-3.5 text-right font-mono">{fmt(item.weight_grams, 2)}g</td>
                      <td className="px-5 py-3.5 text-right font-mono">{fmt(item.price_paid_total)}</td>
                      <td className="px-5 py-3.5 text-right font-mono text-amber-400">
                        {hasValue ? fmt(item.current_value) : '—'}
                      </td>
                      <td className={`px-5 py-3.5 text-right font-mono font-medium ${
                        !hasValue ? 'text-gray-500' : gain >= 0 ? 'text-emerald-400' : 'text-red-400'
                      }`}>
                        {hasValue
                          ? `${gain >= 0 ? '+' : ''}${fmt(gain)} (${fmt(item.gain_loss_pct, 2)}%)`
                          : '—'}
                      </td>
                      <td className="px-5 py-3.5 text-right">
                        <div className="flex justify-end gap-3 text-xs">
                          <button onClick={() => onEdit(item)} className="text-gray-400 transition hover:text-amber-400">
                            Edit
                          </button>
                          <button
                            onClick={() => onDelete(item.id, item.item_name)}
                            className="text-gray-400 transition hover:text-red-400"
                          >
                            Remove
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

function Field({ label, children }) {
  return (
    <div>
      <label className="block text-xs font-medium uppercase tracking-wide text-gray-500">{label}</label>
      <div className="mt-1.5">{children}</div>
    </div>
  );
}

const inputClass =
  'w-full rounded-lg border border-gray-700 bg-gray-950 px-3 py-2 text-sm text-gray-100 placeholder:text-gray-600 focus:border-amber-500 focus:outline-none focus:ring-1 focus:ring-amber-500/30';

function ItemForm({ itemForm, setItemForm, onSubmit, editingId, onCancel, submitting }) {
  const set = (field) => (e) => setItemForm({ ...itemForm, [field]: e.target.value });

  return (
    <div className="max-w-2xl rounded-xl border border-gray-800 bg-gray-900 p-6">
      <div className="mb-6 flex items-center justify-between border-b border-gray-800 pb-4">
        <h2 className="text-lg font-semibold text-gray-100">
          {editingId ? 'Edit purchase' : 'Add a purchase'}
        </h2>
        {editingId && (
          <button onClick={onCancel} className="text-sm text-gray-400 hover:text-gray-200">
            Cancel edit
          </button>
        )}
      </div>

      <form onSubmit={onSubmit} className="space-y-4">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Purchase date">
            <input
              type="date"
              required
              value={itemForm.purchase_date}
              onChange={set('purchase_date')}
              className={inputClass}
            />
          </Field>
          <Field label="Item name">
            <input
              type="text"
              required
              placeholder="e.g. 24K bar, 10g"
              value={itemForm.item_name}
              onChange={set('item_name')}
              className={inputClass}
            />
          </Field>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <Field label="Purity">
            <select value={itemForm.purity_karat} onChange={set('purity_karat')} className={inputClass}>
              {KARAT_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </Field>
          <Field label="Weight (grams)">
            <input
              type="number"
              step="0.001"
              min="0"
              required
              placeholder="0.00"
              value={itemForm.weight_grams}
              onChange={set('weight_grams')}
              className={`${inputClass} font-mono`}
            />
          </Field>
          <Field label="Total paid (BHD)">
            <input
              type="number"
              step="0.001"
              min="0"
              required
              placeholder="0.000"
              value={itemForm.price_paid_total}
              onChange={set('price_paid_total')}
              className={`${inputClass} font-mono`}
            />
          </Field>
        </div>

        <Field label="Vendor (optional)">
          <input
            type="text"
            placeholder="e.g. Local jeweller"
            value={itemForm.vendor}
            onChange={set('vendor')}
            className={inputClass}
          />
        </Field>

        <Field label="Notes (optional)">
          <textarea
            rows="3"
            placeholder="Anything worth remembering about this piece"
            value={itemForm.notes}
            onChange={set('notes')}
            className={inputClass}
          />
        </Field>

        <button
          type="submit"
          disabled={submitting}
          className="w-full rounded-lg bg-amber-500 px-4 py-2.5 text-sm font-semibold text-gray-950 transition hover:bg-amber-400 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {submitting ? 'Saving…' : editingId ? 'Save changes' : 'Add purchase'}
        </button>
      </form>
    </div>
  );
}

function PricesAndSignals({ priceForm, setPriceForm, onSubmit, prices, signals, priceChange, submitting }) {
  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
      <div className="space-y-6 lg:col-span-1">
        {priceChange && (
          <div className="rounded-xl border border-gray-800 bg-gray-900 p-5">
            <p className="text-xs font-medium uppercase tracking-wide text-gray-500">Latest 24K price</p>
            <p className="mt-2 text-2xl font-semibold text-amber-400">
              {fmt(priceChange.latest)} <span className="text-sm font-normal text-gray-500">BHD/g</span>
            </p>
            <p className={`mt-1 text-sm font-medium ${priceChange.diff >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
              {priceChange.diff >= 0 ? '+' : ''}{fmt(priceChange.diff)} ({fmt(priceChange.pct, 2)}%) vs previous
            </p>
          </div>
        )}

        <div className="rounded-xl border border-gray-800 bg-gray-900 p-5">
          <h2 className="mb-4 text-base font-medium text-gray-200">Add a price manually</h2>
          <form onSubmit={onSubmit} className="space-y-4">
            <Field label="Date">
              <input
                type="date"
                required
                value={priceForm.price_date}
                onChange={(e) => setPriceForm({ ...priceForm, price_date: e.target.value })}
                className={inputClass}
              />
            </Field>
            <Field label="24K price per gram (BHD)">
              <input
                type="number"
                step="0.001"
                min="0"
                required
                placeholder="0.000"
                value={priceForm.price_per_gram_24k}
                onChange={(e) => setPriceForm({ ...priceForm, price_per_gram_24k: e.target.value })}
                className={`${inputClass} font-mono`}
              />
            </Field>
            <button
              type="submit"
              disabled={submitting}
              className="w-full rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-2 text-sm font-medium text-amber-400 transition hover:bg-amber-500/20 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {submitting ? 'Saving…' : 'Save price'}
            </button>
          </form>

          <h3 className="mb-2 mt-6 text-xs font-medium uppercase tracking-wide text-gray-500">Recent prices</h3>
          {prices.length === 0 ? (
            <p className="py-6 text-center text-sm text-gray-500">No prices recorded yet.</p>
          ) : (
            <div className="max-h-64 divide-y divide-gray-800 overflow-y-auto rounded-lg border border-gray-800 font-mono text-xs">
              {prices.map((p) => (
                <div key={p.id} className="flex justify-between px-3 py-2">
                  <span className="text-gray-400">{fmtDate(p.price_date)}</span>
                  <span className="text-amber-400">{fmt(p.price_per_gram_24k)} BHD/g</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="rounded-xl border border-gray-800 bg-gray-900 p-5 lg:col-span-2">
        <h2 className="mb-4 text-base font-medium text-gray-200">AI signals</h2>
        {signals.length === 0 ? (
          <div className="py-16 text-center">
            <p className="text-sm text-gray-400">No signals yet.</p>
            <p className="mt-1 text-xs text-gray-500">
              Signals appear here once your n8n workflow starts analyzing the market.
            </p>
          </div>
        ) : (
          <div className="max-h-[600px] space-y-3 overflow-y-auto pr-1">
            {signals.map((s) => (
              <div key={s.id} className="rounded-lg border border-gray-800 bg-gray-950 p-4">
                <div className="mb-2 flex items-center justify-between">
                  <span
                    className={`rounded px-2 py-0.5 text-xs font-semibold ${
                      s.signal_type === 'BUY'
                        ? 'bg-emerald-500/10 text-emerald-400'
                        : s.signal_type === 'SELL'
                        ? 'bg-red-500/10 text-red-400'
                        : 'bg-gray-800 text-gray-400'
                    }`}
                  >
                    {s.signal_type}
                  </span>
                  <span className="font-mono text-xs text-gray-500">
                    {new Date(s.signal_date).toLocaleString()}
                  </span>
                </div>
                <p className="text-sm leading-relaxed text-gray-300">{s.reasoning}</p>
                {s.price_at_signal && (
                  <p className="mt-2 font-mono text-xs text-gray-500">
                    24K price at signal: {fmt(s.price_at_signal)} BHD/g
                  </p>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
