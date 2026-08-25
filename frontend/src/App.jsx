import { useState, useCallback } from 'react';
import { apiRequest } from './api/client.js';
import { useAuth } from './auth/useAuth.js';
import { LoginPage } from './auth/LoginPage.jsx';
import { useGoldData } from './hooks/useGoldData.js';
import { useSignalRun } from './hooks/useSignalRun.js';
import { AppShell } from './components/layout/AppShell.jsx';
import { StatGrid } from './components/holdings/StatGrid.jsx';
import { HoldingsTable } from './components/holdings/HoldingsTable.jsx';
import { ItemForm } from './components/forms/ItemForm.jsx';
import { PriceForm } from './components/forms/PriceForm.jsx';
import { PriceChart } from './components/market/PriceChart.jsx';
import { PriceHistoryList } from './components/market/PriceHistoryList.jsx';
import { SignalPanel } from './components/signals/SignalPanel.jsx';
import { Card } from './components/ui/Card.jsx';
import { Toast } from './components/ui/Toast.jsx';

function Dashboard({ onSignOut }) {
  const [activeTab, setActiveTab] = useState('holdings');
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
  const spotPrice = prices[0]?.price_per_gram_24k;
  const spotDate = prices[0]?.price_date;
  const purchaseMarks = items.map((item) => ({
    date: item.purchase_date,
    pricePerGram: item.price_per_gram_paid,
  }));

  return (
    <AppShell
      activeTab={activeTab}
      onTabChange={setActiveTab}
      spotPrice={spotPrice}
      spotDate={spotDate}
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
            <Card title="24K spot price">
              <PriceChart prices={prices} purchases={purchaseMarks} />
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
            <PriceHistoryList prices={prices} />
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
