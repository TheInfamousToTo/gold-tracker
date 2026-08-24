import { NavTabs } from './NavTabs.jsx';
import { fmt } from '../../lib/format.js';

/**
 * The masthead doubles as the price board: the current 24K spot is the
 * number the owner opens this app to see, so it is the first thing on
 * the page rather than one card among four.
 */
export function AppShell({ activeTab, onTabChange, spotPrice, error, onReconnect, children }) {
  return (
    <div className="min-h-screen bg-ink">
      <header className="sticky top-0 z-40 border-b border-line bg-ink/95 backdrop-blur">
        <div className="mx-auto flex max-w-7xl flex-wrap items-center gap-x-8 gap-y-3 px-4 py-3 sm:px-6 lg:px-8">
          <div className="flex items-baseline gap-3">
            <span className="font-display text-base font-bold uppercase tracking-stamp text-chalk">
              Gold Tracker
            </span>
          </div>

          <div className="flex items-baseline gap-2">
            <span className="stamp">24K spot</span>
            <span className="font-mono text-lg font-semibold text-gold-400">
              {spotPrice != null ? fmt(spotPrice, 3) : '—'}
            </span>
            <span className="stamp">BHD/g</span>
          </div>

          <div className="ml-auto">
            <NavTabs activeTab={activeTab} onChange={onTabChange} />
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {error && (
          <div className="mb-6 flex items-center justify-between gap-4 rounded-lg border border-oxide/40 bg-oxide/10 px-4 py-3">
            <p className="text-sm text-oxide">Can't reach the API: {error}</p>
            <button
              onClick={onReconnect}
              className="font-display text-[10px] font-semibold uppercase tracking-stamp text-oxide hover:text-chalk"
            >
              Retry
            </button>
          </div>
        )}
        {children}
      </main>
    </div>
  );
}
