import { useState, useCallback } from 'react';
import { apiRequest } from './api/client.js';
import { useAuth } from './auth/useAuth.js';
import { LoginPage } from './auth/LoginPage.jsx';
import { useGoldData } from './hooks/useGoldData.js';
import { spotOf, METAL_FINE_LABEL } from './lib/prices.js';
import { useSignalRun } from './hooks/useSignalRun.js';
import { AppShell } from './components/layout/AppShell.jsx';
import { StatGrid } from './components/holdings/StatGrid.jsx';
import { HoldingsTable } from './components/holdings/HoldingsTable.jsx';
import { MassByPurity } from './components/holdings/MassByPurity.jsx';
import { ItemForm } from './components/forms/ItemForm.jsx';
import { PriceForm } from './components/forms/PriceForm.jsx';
import { PriceChart } from './components/market/PriceChart.jsx';
import { MetalToggle } from './components/market/MetalToggle.jsx';
import { PriceHistoryList } from './components/market/PriceHistoryList.jsx';
import { SignalPanel } from './components/signals/SignalPanel.jsx';
import { Card } from './components/ui/Card.jsx';
import { Toast } from './components/ui/Toast.jsx';

function Dashboard({ onSignOut }) {
  const [activeTab, setActiveTab] = useState('holdings');
  const [marketMetal, setMarketMetal] = useState('gold');
  const [editingItem, setEditingItem] = useState(null);
  const [toast, setToast] = useState(null);

  const { portfolio, prices, signals, loading, error, refreshData } = useGoldData();
  const signalRun = useSignalRun(refreshData);

  const showToast = useCallback((message, kind = 'success') => {
    setToast({ message, kind });
    setTimeout(() => setToast(null), 4000);
  }, []);

  const deleteItem = useCallback(
    async (item) => {
      if (!confirm(`Delete ${item.item_name}? This can't be undone.`)) return;
      try {
        await apiRequest(`/api/items/${item.id}`, { method: 'DELETE' });
        showToast(`Deleted ${item.item_name}`);
        await refreshData();
      } catch (err) {
        showToast(err.message, 'error');
      }
    },
    [refreshData, showToast],
  );

  const editItem = useCallback((item) => {
    setEditingItem(item);
    setActiveTab('add-item');
  }, []);

  const handleItemSaved = useCallback(async () => {
    showToast(editingItem ? 'Changes saved' : 'Purchase added');
    setEditingItem(null);
    await refreshData();
    setActiveTab('holdings');
  }, [editingItem, refreshData, showToast]);

  const items = portfolio.items || [];

  const spots = {
    gold: { price: spotOf(prices.gold), date: prices.gold[0]?.price_date },
    silver: { price: spotOf(prices.silver), date: prices.silver[0]?.price_date },
  };

  // The chart marks purchases against the metal it is showing, so a
  // silver piece must not appear on the gold series at a rate that
  // would read as wildly underwater.
  const metalPrices = prices[marketMetal] || [];
  const purchaseMarks = items
    .filter((item) => (item.metal_type || 'gold') === marketMetal)
    .map((item) => ({
      date: item.purchase_date,
      pricePerGram: item.price_per_gram_paid,
    }));

  return (
    <AppShell
      activeTab={activeTab}
      onTabChange={setActiveTab}
      spots={spots}
      error={error}
      onReconnect={refreshData}
      onSignOut={onSignOut}
    >
      {activeTab === 'holdings' && (
        <div className="space-y-6">
          <StatGrid totals={portfolio.totals || {}} loading={loading} />
          <HoldingsTable
            items={items}
            loading={loading}
            onEdit={editItem}
            onDelete={deleteItem}
            onAddFirst={() => setActiveTab('add-item')}
          />
          <MassByPurity items={items} loading={loading} />
        </div>
      )}

      {activeTab === 'add-item' && (
        <div className="mx-auto max-w-3xl">
          <ItemForm
            editingItem={editingItem}
            onSaved={handleItemSaved}
            onCancelEdit={() => setEditingItem(null)}
          />
        </div>
      )}

      {activeTab === 'market' && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className="space-y-6 lg:col-span-2">
            <Card
              title={`${METAL_FINE_LABEL[marketMetal]} spot price`}
              actions={<MetalToggle metal={marketMetal} onChange={setMarketMetal} />}
            >
              <PriceChart prices={metalPrices} purchases={purchaseMarks} />
            </Card>
            <SignalPanel
              signals={signals}
              status={signalRun.status}
              generating={signalRun.generating}
              error={signalRun.error}
              onGenerate={signalRun.generate}
            />
          </div>
          <div className="space-y-6">
            <PriceForm onSaved={refreshData} />
            <PriceHistoryList
              prices={metalPrices}
              title={`Recent ${marketMetal} prices`}
            />
          </div>
        </div>
      )}

      <Toast message={toast?.message} kind={toast?.kind} onDismiss={() => setToast(null)} />
    </AppShell>
  );
}

/**
 * Nothing about the portfolio renders until the viewer has a session.
 * The dashboard is mounted only when authenticated, so its data hooks
 * never fire requests that would come back 401.
 */
export default function App() {
  const { checking, authenticated, loginConfigured, signIn, signOut } = useAuth();

  if (checking) {
    return <div className="min-h-screen bg-ink" aria-busy="true" />;
  }

  if (!authenticated) {
    return <LoginPage onSignIn={signIn} loginConfigured={loginConfigured} />;
  }

  return <Dashboard onSignOut={signOut} />;
}
