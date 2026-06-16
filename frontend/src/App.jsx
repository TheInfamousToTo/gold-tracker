import React, { useState, useEffect, useCallback, useMemo } from 'react';

// --- Constants & Types ---

const KARAT_OPTIONS = [
  { value: '24', label: '24K (99.9%)', description: 'Investment Grade' },
  { value: '22', label: '22K (91.6%)', description: 'Traditional Jewelry' },
  { value: '21', label: '21K (87.5%)', description: 'Common GCC Standard' },
  { value: '18', label: '18K (75.0%)', description: 'Fine Jewelry' },
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
  { id: 'holdings', label: 'Holdings', icon: '📊' },
  { id: 'add-item', label: 'Add Purchase', icon: '➕' },
  { id: 'prices', label: 'Market & Signals', icon: '📈' },
];

// --- Utilities ---

const fmt = (value, decimals = 3) => {
  const n = Number(value);
  return Number.isFinite(n) ? n.toLocaleString(undefined, { minimumFractionDigits: decimals, maximumFractionDigits: decimals }) : '0.000';
};

const fmtDate = (value) => {
  if (!value) return '—';
  return new Date(value).toLocaleDateString(undefined, { 
    year: 'numeric', month: 'short', day: 'numeric' 
  });
};

async function apiRequest(url, options) {
  const res = await fetch(url, options);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(body.error || `Request failed (${res.status})`);
  return body;
}

// --- Components ---

function Badge({ children, variant = 'default' }) {
  const variants = {
    default: 'bg-gray-800 text-gray-300',
    success: 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20',
    danger: 'bg-red-500/10 text-red-400 border border-red-500/20',
    warning: 'bg-amber-500/10 text-amber-400 border border-amber-500/20',
    info: 'bg-blue-500/10 text-blue-400 border border-blue-500/20',
  };
  return (
    <span className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider ${variants[variant]}`}>
      {children}
    </span>
  );
}

function StatCard({ label, value, unit, trend, isLoading }) {
  return (
    <div className="bg-gray-900/50 backdrop-blur-sm border border-white/5 rounded-2xl p-6 transition-all hover:border-white/10 group">
      <div className="flex justify-between items-start">
        <p className="text-xs font-semibold text-gray-500 uppercase tracking-widest">{label}</p>
        {trend && (
          <span className={`text-xs font-bold ${trend >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
            {trend >= 0 ? '↑' : '↓'} {Math.abs(trend).toFixed(2)}%
          </span>
        )}
      </div>
      <div className="mt-4 flex items-baseline gap-1">
        {isLoading ? (
          <div className="h-8 w-32 bg-gray-800 animate-pulse rounded" />
        ) : (
          <>
            <span className="text-3xl font-bold text-white tracking-tight">{value}</span>
            <span className="text-sm font-medium text-gray-500">{unit}</span>
          </>
        )}
      </div>
    </div>
  );
}

// --- Main Application ---

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
    try {
      const [portfolioData, pricesData, signalsData] = await Promise.all([
        apiRequest('/api/portfolio'),
        apiRequest('/api/prices'),
        apiRequest('/api/signals'),
      ]);
      setPortfolio(portfolioData);
      setPrices(pricesData || []);
      setSignals(signalsData || []);
      setError(null);
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
      const method = editingId ? 'PUT' : 'POST';
      const url = editingId ? `/api/items/${editingId}` : '/api/items';
      await apiRequest(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(itemForm),
      });
      showToast(editingId ? 'Position updated' : 'Asset added successfully');
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

  const deleteItem = async (id, name) => {
    if (!confirm(`Are you sure you want to remove ${name}?`)) return;
    try {
      await apiRequest(`/api/items/${id}`, { method: 'DELETE' });
      showToast('Position liquidated');
      await refreshData();
    } catch (err) {
      showToast(err.message, 'error');
    }
  };

  const totals = portfolio.totals || {};
  const isGain = (totals.total_gain_loss || 0) >= 0;

  return (
    <div className="min-h-screen bg-[#0a0a0b] text-gray-200 selection:bg-amber-500/30">
      {/* Navigation */}
      <nav className="border-b border-white/5 bg-black/20 backdrop-blur-xl sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-20 items-center">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-gradient-to-br from-amber-400 to-amber-600 rounded-xl flex items-center justify-center shadow-lg shadow-amber-500/20">
                <span className="text-xl">💰</span>
              </div>
              <div>
                <h1 className="text-lg font-bold text-white leading-tight">GoldTracker</h1>
                <p className="text-[10px] text-gray-500 uppercase tracking-widest font-bold">Enterprise Assets</p>
              </div>
            </div>
            
            <div className="flex bg-gray-900/50 p-1 rounded-xl border border-white/5">
              {TABS.map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`px-4 py-2 rounded-lg text-sm font-semibold transition-all flex items-center gap-2 ${
                    activeTab === tab.id
                      ? 'bg-amber-500 text-black shadow-lg shadow-amber-500/20'
                      : 'text-gray-400 hover:text-white hover:bg-white/5'
                  }`}
                >
                  <span className="text-base">{tab.icon}</span>
                  <span className="hidden sm:inline">{tab.label}</span>
                </button>
              ))}
            </div>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
        {error && (
          <div className="mb-8 p-4 bg-red-500/10 border border-red-500/20 rounded-2xl flex items-center justify-between">
            <div className="flex items-center gap-3 text-red-400">
              <span className="text-xl">⚠️</span>
              <p className="text-sm font-medium">Connectivity Issue: {error}</p>
            </div>
            <button onClick={refreshData} className="text-xs font-bold uppercase tracking-widest text-red-400 hover:text-red-300 transition-colors">
              Reconnect
            </button>
          </div>
        )}

        {activeTab === 'holdings' && (
          <div className="space-y-10 animate-in fade-in slide-in-from-bottom-4 duration-700">
            {/* Summary Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
              <StatCard label="Total Investment" value={fmt(totals.total_paid, 2)} unit="BHD" isLoading={loading} />
              <StatCard label="Net Valuation" value={fmt(totals.total_value, 2)} unit="BHD" isLoading={loading} />
              <StatCard 
                label="Unrealized P/L" 
                value={`${isGain ? '+' : ''}${fmt(totals.total_gain_loss, 2)}`} 
                unit="BHD" 
                trend={totals.total_gain_loss_pct}
                isLoading={loading}
              />
              <StatCard label="Total Return" value={`${isGain ? '+' : ''}${fmt(totals.total_gain_loss_pct, 2)}`} unit="%" isLoading={loading} />
            </div>

            {/* Asset Table */}
            <div className="bg-gray-900/30 border border-white/5 rounded-3xl overflow-hidden shadow-2xl">
              <div className="px-8 py-6 border-b border-white/5 flex justify-between items-center bg-white/5">
                <h2 className="text-sm font-bold uppercase tracking-widest text-gray-400">Portfolio Inventory</h2>
                <div className="flex items-center gap-2">
                  <div className={`w-2 h-2 rounded-full ${loading ? 'bg-amber-500 animate-pulse' : 'bg-emerald-500'}`} />
                  <span className="text-[10px] font-bold text-gray-500 uppercase tracking-tighter">Live Status</span>
                </div>
              </div>
              
              <div className="overflow-x-auto">
                <table className="w-full text-left">
                  <thead>
                    <tr className="text-[10px] font-bold text-gray-500 uppercase tracking-widest bg-black/40">
                      <th className="px-8 py-4">Asset Identification</th>
                      <th className="px-8 py-4">Purity</th>
                      <th className="px-8 py-4 text-right">Mass (g)</th>
                      <th className="px-8 py-4 text-right">Entry Price</th>
                      <th className="px-8 py-4 text-right">Acquisition</th>
                      <th className="px-8 py-4 text-right">Market Value</th>
                      <th className="px-8 py-4 text-right">Performance</th>
                      <th className="px-8 py-4"></th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-white/5">
                    {portfolio.items.length > 0 ? portfolio.items.map((item) => (
                      <tr key={item.id} className="hover:bg-white/5 transition-colors group">
                        <td className="px-8 py-6">
                          <p className="text-sm font-bold text-white group-hover:text-amber-400 transition-colors">{item.item_name}</p>
                          <p className="text-[10px] text-gray-500 mt-1 uppercase font-bold tracking-tighter">{item.vendor || 'Unknown Source'} • {fmtDate(item.purchase_date)}</p>
                        </td>
                        <td className="px-8 py-6">
                          <Badge variant={item.purity_karat >= 22 ? 'success' : 'default'}>{item.purity_karat}K Standard</Badge>
                        </td>
                        <td className="px-8 py-6 text-right font-mono text-sm text-gray-300">{fmt(item.weight_grams, 2)}</td>
                        <td className="px-8 py-6 text-right font-mono text-sm text-amber-500/80">{fmt(item.price_per_gram_paid, 3)}</td>
                        <td className="px-8 py-6 text-right font-mono text-sm text-gray-300">{fmt(item.price_paid_total, 2)}</td>
                        <td className="px-8 py-6 text-right font-mono text-sm text-white">{fmt(item.current_value, 2)}</td>
                        <td className="px-8 py-6 text-right">
                          <p className={`text-sm font-bold ${item.gain_loss >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
                            {item.gain_loss >= 0 ? '+' : ''}{fmt(item.gain_loss, 2)}
                          </p>
                          <p className={`text-[10px] font-bold ${item.gain_loss >= 0 ? 'text-emerald-500/50' : 'text-red-500/50'}`}>
                            {fmt(item.gain_loss_pct, 2)}% Yield
                          </p>
                        </td>
                        <td className="px-8 py-6 text-right">
                          <button 
                            onClick={() => deleteItem(item.id, item.item_name)}
                            className="text-gray-600 hover:text-red-400 transition-colors p-2 hover:bg-red-500/10 rounded-lg"
                          >
                            🗑️
                          </button>
                        </td>
                      </tr>
                    )) : (
                      <tr>
                        <td colSpan="7" className="px-8 py-20 text-center">
                          <div className="max-w-xs mx-auto space-y-4">
                            <span className="text-4xl grayscale opacity-50 block">📭</span>
                            <p className="text-sm text-gray-500 font-medium">Vault is currently empty. Start logging your assets to begin tracking performance.</p>
                            <button onClick={() => setActiveTab('add-item')} className="text-xs font-bold text-amber-500 uppercase tracking-widest hover:text-amber-400">Initialize First Asset</button>
                          </div>
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'add-item' && (
          <div className="max-w-2xl mx-auto animate-in fade-in zoom-in-95 duration-500">
            <div className="bg-gray-900/50 border border-white/5 rounded-3xl p-10 space-y-8 shadow-2xl">
              <div>
                <h2 className="text-2xl font-bold text-white">Asset Registration</h2>
                <p className="text-sm text-gray-500 mt-2">Document your acquisition with precise technical specifications.</p>
              </div>

              <form onSubmit={handleItemSubmit} className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <label className="text-[10px] font-bold text-gray-500 uppercase tracking-widest">Date of Purchase</label>
                    <input 
                      type="date" required value={itemForm.purchase_date}
                      onChange={(e) => setItemForm({...itemForm, purchase_date: e.target.value})}
                      className="w-full bg-black/40 border border-white/5 rounded-xl px-4 py-3 text-sm focus:border-amber-500/50 focus:outline-none transition-all"
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-[10px] font-bold text-gray-500 uppercase tracking-widest">Asset Name</label>
                    <input 
                      type="text" required placeholder="Swiss 10g Gold Bar" value={itemForm.item_name}
                      onChange={(e) => setItemForm({...itemForm, item_name: e.target.value})}
                      className="w-full bg-black/40 border border-white/5 rounded-xl px-4 py-3 text-sm focus:border-amber-500/50 focus:outline-none transition-all"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  <div className="space-y-2">
                    <label className="text-[10px] font-bold text-gray-500 uppercase tracking-widest">Purity (Karat)</label>
                    <select 
                      value={itemForm.purity_karat} onChange={(e) => setItemForm({...itemForm, purity_karat: e.target.value})}
                      className="w-full bg-black/40 border border-white/5 rounded-xl px-4 py-3 text-sm focus:border-amber-500/50 focus:outline-none transition-all appearance-none"
                    >
                      {KARAT_OPTIONS.map(opt => <option key={opt.value} value={opt.value} className="bg-gray-900">{opt.label}</option>)}
                    </select>
                  </div>
                  <div className="space-y-2">
                    <label className="text-[10px] font-bold text-gray-500 uppercase tracking-widest">Net Mass (g)</label>
                    <input 
                      type="number" step="0.001" required placeholder="0.000" value={itemForm.weight_grams}
                      onChange={(e) => setItemForm({...itemForm, weight_grams: e.target.value})}
                      className="w-full bg-black/40 border border-white/5 rounded-xl px-4 py-3 text-sm focus:border-amber-500/50 focus:outline-none transition-all font-mono"
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-[10px] font-bold text-gray-500 uppercase tracking-widest">Gross Price (BHD)</label>
                    <input 
                      type="number" step="0.001" required placeholder="0.000" value={itemForm.price_paid_total}
                      onChange={(e) => setItemForm({...itemForm, price_paid_total: e.target.value})}
                      className="w-full bg-black/40 border border-white/5 rounded-xl px-4 py-3 text-sm focus:border-amber-500/50 focus:outline-none transition-all font-mono"
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <label className="text-[10px] font-bold text-gray-500 uppercase tracking-widest">Notes & Details</label>
                  <textarea 
                    rows="3" placeholder="Additional identifying marks, certificate numbers..." value={itemForm.notes}
                    onChange={(e) => setItemForm({...itemForm, notes: e.target.value})}
                    className="w-full bg-black/40 border border-white/5 rounded-xl px-4 py-3 text-sm focus:border-amber-500/50 focus:outline-none transition-all resize-none"
                  />
                </div>

                <button 
                  type="submit" disabled={submitting}
                  className="w-full bg-amber-500 text-black font-bold py-4 rounded-xl shadow-xl shadow-amber-500/10 hover:bg-amber-400 transition-all active:scale-[0.98] disabled:opacity-50"
                >
                  {submitting ? 'Processing Transaction...' : 'Confirm Asset Registration'}
                </button>
              </form>
            </div>
          </div>
        )}

        {activeTab === 'prices' && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 animate-in fade-in duration-700">
            {/* Market Data */}
            <div className="lg:col-span-1 space-y-8">
              <div className="bg-gray-900/50 border border-white/5 rounded-3xl p-8 space-y-6 shadow-xl">
                <h3 className="text-sm font-bold uppercase tracking-widest text-gray-400">Manual Price Entry</h3>
                <form onSubmit={(e) => { e.preventDefault(); handlePriceSubmit(e); }} className="space-y-4">
                  <div className="space-y-2">
                    <label className="text-[10px] font-bold text-gray-500 uppercase tracking-widest">Valuation Date</label>
                    <input 
                      type="date" required value={priceForm.price_date}
                      onChange={(e) => setPriceForm({...priceForm, price_date: e.target.value})}
                      className="w-full bg-black/40 border border-white/5 rounded-xl px-4 py-3 text-sm"
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-[10px] font-bold text-gray-500 uppercase tracking-widest">24K Spot Price (BHD/g)</label>
                    <input 
                      type="number" step="0.001" required placeholder="0.000" value={priceForm.price_per_gram_24k}
                      onChange={(e) => setPriceForm({...priceForm, price_per_gram_24k: e.target.value})}
                      className="w-full bg-black/40 border border-white/5 rounded-xl px-4 py-3 text-sm font-mono"
                    />
                  </div>
                  <button className="w-full bg-white/5 border border-white/10 text-white font-bold py-3 rounded-xl hover:bg-white/10 transition-all">
                    Update Market Spot
                  </button>
                </form>
              </div>

              <div className="bg-gray-900/50 border border-white/5 rounded-3xl overflow-hidden shadow-xl">
                <div className="px-6 py-4 border-b border-white/5 bg-white/5">
                  <h3 className="text-[10px] font-bold uppercase tracking-widest text-gray-400">Historical Benchmarks</h3>
                </div>
                <div className="divide-y divide-white/5 max-h-96 overflow-y-auto">
                  {prices.map(p => (
                    <div key={p.id} className="px-6 py-4 flex justify-between items-center group hover:bg-white/5 transition-colors">
                      <span className="text-[10px] font-bold text-gray-500 uppercase tracking-tighter">{fmtDate(p.price_date)}</span>
                      <span className="text-sm font-mono text-amber-400 font-bold">{fmt(p.price_per_gram_24k, 3)}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* AI Signals */}
            <div className="lg:col-span-2 space-y-8">
              <div className="bg-gray-900/30 border border-white/5 rounded-3xl p-10 shadow-2xl min-h-[600px]">
                <div className="flex justify-between items-center mb-10">
                  <h2 className="text-2xl font-bold text-white flex items-center gap-3">
                    <span className="text-amber-500">⚡</span> Artificial Intelligence Signals
                  </h2>
                  <Badge variant="info">Neural Analysis Active</Badge>
                </div>

                {signals.length > 0 ? (
                  <div className="space-y-6">
                    {signals.map(s => (
                      <div key={s.id} className="bg-white/5 border border-white/5 rounded-2xl p-6 hover:border-white/10 transition-all">
                        <div className="flex justify-between items-start mb-4">
                          <div className={`px-4 py-1.5 rounded-full text-[10px] font-black uppercase tracking-[0.2em] ${
                            s.signal_type === 'BUY' ? 'bg-emerald-500/20 text-emerald-400' : 
                            s.signal_type === 'SELL' ? 'bg-red-500/20 text-red-400' : 'bg-gray-800 text-gray-400'
                          }`}>
                            Recommendation: {s.signal_type}
                          </div>
                          <span className="text-[10px] font-bold text-gray-600 uppercase tracking-widest">{new Date(s.signal_date).toLocaleString()}</span>
                        </div>
                        <p className="text-gray-300 text-sm leading-relaxed font-medium">{s.reasoning}</p>
                        {s.price_at_signal && (
                          <div className="mt-4 pt-4 border-t border-white/5 flex items-center gap-2">
                            <span className="text-[10px] font-bold text-gray-500 uppercase tracking-widest">Entry Benchmark:</span>
                            <span className="text-xs font-mono font-bold text-amber-500">{fmt(s.price_at_signal, 3)} BHD/g</span>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="h-[400px] flex flex-col items-center justify-center text-center space-y-6 grayscale opacity-30">
                    <span className="text-6xl">🧠</span>
                    <div className="max-w-xs space-y-2">
                      <p className="text-sm font-bold text-white">Awaiting Neural Processing</p>
                      <p className="text-[10px] font-medium text-gray-500 uppercase tracking-widest leading-loose">Automated market analysis will manifest here once n8n heuristics complete their cycle.</p>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </main>

      {/* Toast Notification */}
      {toast && (
        <div className={`fixed bottom-8 right-8 z-[100] animate-in fade-in slide-in-from-right-10 duration-500 ${
          toast.kind === 'error' ? 'bg-red-500' : 'bg-emerald-500'
        } text-black font-black text-[10px] uppercase tracking-widest px-6 py-4 rounded-2xl shadow-2xl flex items-center gap-3`}>
          <span>{toast.kind === 'error' ? '❌' : '✅'}</span>
          {toast.message}
        </div>
      )}
    </div>
  );
}
