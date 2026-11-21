import { NavTabs, BottomNav } from './NavTabs.jsx';
import { fmt, fmtDate, daysSince } from '../../lib/format.js';
import { METAL_FINE_LABEL } from '../../lib/prices.js';

/** A price older than this is no longer the price. */
const STALE_AFTER_DAYS = 2;

function feedState(price, date) {
  const age = daysSince(date);
  if (price == null || age == null) return { className: 'andon-bad', label: 'No price' };
  if (age > STALE_AFTER_DAYS) return { className: 'andon-warn', label: `Stale · ${age}d` };
  return { className: 'andon-ok', label: 'Current' };
}

/**
 * The spot price for one metal, with a chip saying whether the board
 * can be trusted.
 *
 * The figure itself is set in neutral type. A spot price is data, not a
 * verdict — colouring it would spend the andon on something that is
 * never good or bad. What does get a colour is the feed's condition:
 * fresh, stale, or missing. Each metal carries its own, because the
 * gold feed can be running while the silver one has stopped.
 */
function SpotPrice({ metal, price, date }) {
  const feed = feedState(price, date);
  return (
    <div className="flex shrink-0 items-baseline gap-1.5 sm:gap-2">
      <span className="stamp">
        {METAL_FINE_LABEL[metal]}<span className="hidden sm:inline"> {metal}</span>
      </span>
      <span className="font-mono text-base font-semibold text-chalk sm:text-lg">
        {price != null ? fmt(price, 3) : '—'}
      </span>
      <span className="stamp hidden sm:inline">BHD/g</span>
      <span
        className={`andon ${feed.className} sm:ml-1`}
        title={date ? `Last recorded ${fmtDate(date)}` : 'No price recorded'}
      >
        {feed.label}
      </span>
    </div>
  );
}

/**
 * The masthead doubles as the price board: the current spot is the
 * number the owner opens this app to see, so it is the first thing on
 * the page rather than one card among four.
 *
 * Silver appears only once there is a silver price to show. An owner
 * who only holds gold should not be reading around a permanent dash.
 */
export function AppShell({ activeTab, onTabChange, spots, error, onReconnect, onSignOut, children }) {
  const gold = spots?.gold || {};
  const silver = spots?.silver || {};

  return (
    <div className="min-h-screen bg-ink">
      <header className="sticky top-0 z-40 border-b border-line bg-ink/95 pt-[env(safe-area-inset-top)] backdrop-blur">
        <div className="mx-auto flex max-w-7xl flex-wrap items-center gap-x-6 gap-y-2 px-4 py-2.5 sm:gap-x-8 sm:px-6 sm:py-3 lg:px-8">
          <div className="flex items-center gap-2.5">
            <span aria-hidden="true" className="h-3.5 w-3.5 rounded-[2px] bg-brand" />
            <span className="font-display text-sm font-bold uppercase tracking-stamp text-chalk sm:text-base">
              Gold Tracker
            </span>
          </div>

          {/* On a phone the spot prices sit on their own line and scroll
              sideways rather than wrapping into a second and third row. */}
          <div className="order-last -mx-4 flex w-[calc(100%+2rem)] gap-5 overflow-x-auto px-4 sm:order-none sm:mx-0 sm:w-auto sm:gap-8 sm:overflow-visible sm:px-0">
            <SpotPrice metal="gold" price={gold.price} date={gold.date} />
            {silver.price != null && (
              <SpotPrice metal="silver" price={silver.price} date={silver.date} />
            )}
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

      <main className="mx-auto max-w-7xl px-4 pb-28 pt-4 sm:px-6 sm:py-8 lg:px-8">
        {error && (
          <div className="mb-6 flex flex-wrap items-center justify-between gap-x-4 gap-y-2 rounded-lg border-l-4 border-l-bad border-y border-r border-bad/30 bg-bad/10 px-4 py-3">
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
      <BottomNav activeTab={activeTab} onChange={onTabChange} />
    </div>
  );
}
