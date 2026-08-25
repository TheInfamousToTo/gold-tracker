import { NavTabs } from './NavTabs.jsx';
import { fmt, fmtDate, daysSince } from '../../lib/format.js';

/** A price older than this is no longer the price. */
const STALE_AFTER_DAYS = 2;

/**
 * The masthead doubles as the price board: the current 24K spot is the
 * number the owner opens this app to see, so it is the first thing on
 * the page rather than one card among four.
 *
 * The figure itself is set in neutral type. A spot price is data, not a
 * verdict — colouring it would spend the andon on something that is
 * never good or bad. What does get a colour is whether the board can be
 * trusted: fresh, stale, or missing.
 */
export function AppShell({ activeTab, onTabChange, spotPrice, spotDate, error, onReconnect, onSignOut, children }) {
  const age = daysSince(spotDate);
  const feed =
    spotPrice == null || age == null
      ? { className: 'andon-bad', label: 'No price' }
      : age > STALE_AFTER_DAYS
        ? { className: 'andon-warn', label: `Stale · ${age}d` }
        : { className: 'andon-ok', label: 'Current' };

  return (
    <div className="min-h-screen bg-ink">
      <header className="sticky top-0 z-40 border-b border-line bg-ink/95 backdrop-blur">
        <div className="mx-auto flex max-w-7xl flex-wrap items-center gap-x-8 gap-y-3 px-4 py-3 sm:px-6 lg:px-8">
          <div className="flex items-center gap-2.5">
            <span aria-hidden="true" className="h-3.5 w-3.5 rounded-[2px] bg-brand" />
            <span className="font-display text-base font-bold uppercase tracking-stamp text-chalk">
              Gold Tracker
            </span>
          </div>

          <div className="flex items-baseline gap-2">
            <span className="stamp">24K spot</span>
            <span className="font-mono text-lg font-semibold text-chalk">
              {spotPrice != null ? fmt(spotPrice, 3) : '—'}
            </span>
            <span className="stamp">BHD/g</span>
            <span
              className={`andon ${feed.className} ml-1`}
              title={spotDate ? `Last recorded ${fmtDate(spotDate)}` : 'No price recorded'}
            >
              {feed.label}
            </span>
          </div>

          <div className="ml-auto flex items-center gap-2">
            <NavTabs activeTab={activeTab} onChange={onTabChange} />
            {onSignOut && (
              <button
                onClick={onSignOut}
                className="rounded-chip px-3 py-2 font-display text-[11px] font-semibold uppercase tracking-stamp text-muted transition-colors hover:bg-line/40 hover:text-chalk"
              >
                Sign out
              </button>
            )}
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {error && (
          <div className="mb-6 flex items-center justify-between gap-4 rounded-lg border-l-4 border-l-bad border-y border-r border-bad/30 bg-bad/10 px-4 py-3">
            <p className="flex items-center gap-2 text-sm text-bad-bright">
              <span className="andon andon-bad">Down</span>
              Can't reach the API: {error}
            </p>
            <button
              onClick={onReconnect}
              className="font-display text-[10px] font-semibold uppercase tracking-stamp text-bad-bright hover:text-chalk"
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
