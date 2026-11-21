export const TABS = [
  { id: 'holdings', label: 'Holdings', short: 'Holdings', icon: 'M4 6h16M4 12h16M4 18h10' },
  { id: 'add-item', label: 'Add physical', short: 'Add', icon: 'M12 5v14M5 12h14' },
  { id: 'stocks', label: 'Stocks & ETFs', short: 'Stocks', icon: 'M4 18l5-6 4 3 7-9' },
  { id: 'market', label: 'Market', short: 'Market', icon: 'M4 20V10M10 20V4M16 20v-7M22 20H2' },
  { id: 'reports', label: 'Reports', short: 'Reports', icon: 'M7 3h7l5 5v13H7zM14 3v5h5M10 13h6M10 17h6' },
];

/** Navigation is neutral: where you are is not a good or bad condition. */
export function NavTabs({ activeTab, onChange }) {
  return (
    <nav className="hidden gap-1 sm:flex" aria-label="Sections">
      {TABS.map((tab) => {
        const active = activeTab === tab.id;
        return (
          <button
            key={tab.id}
            onClick={() => onChange(tab.id)}
            aria-current={active ? 'page' : undefined}
            className={`rounded-chip px-3 py-2 font-display text-[11px] font-semibold uppercase tracking-stamp transition-colors ${
              active ? 'bg-chalk text-ink' : 'text-muted hover:bg-line/40 hover:text-chalk'
            }`}
          >
            {tab.label}
          </button>
        );
      })}
    </nav>
  );
}

/**
 * Phones get a thumb-reach tab bar pinned to the bottom instead of a
 * row of tabs that wraps under the masthead.
 */
export function BottomNav({ activeTab, onChange }) {
  return (
    <nav
      aria-label="Sections"
      className="fixed inset-x-0 bottom-0 z-40 grid grid-cols-5 border-t border-line bg-ink/95 pb-[env(safe-area-inset-bottom)] backdrop-blur sm:hidden"
    >
      {TABS.map((tab) => {
        const active = activeTab === tab.id;
        return (
          <button
            key={tab.id}
            onClick={() => onChange(tab.id)}
            aria-current={active ? 'page' : undefined}
            className={`flex min-h-[56px] flex-col items-center justify-center gap-1 font-display text-[10px] font-semibold uppercase tracking-wide ${
              active ? 'text-chalk' : 'text-muted'
            }`}
          >
            <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d={tab.icon} />
            </svg>
            {tab.short}
            <span aria-hidden="true" className={`h-0.5 w-6 rounded-full ${active ? 'bg-chalk' : 'bg-transparent'}`} />
          </button>
        );
      })}
    </nav>
  );
}
