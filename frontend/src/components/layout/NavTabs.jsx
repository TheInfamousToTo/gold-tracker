export const TABS = [
  { id: 'holdings', label: 'Holdings' },
  { id: 'add-item', label: 'Add purchase' },
  { id: 'market', label: 'Market' },
];

export function NavTabs({ activeTab, onChange }) {
  return (
    <nav className="flex gap-1" aria-label="Sections">
      {TABS.map((tab) => {
        const active = activeTab === tab.id;
        return (
          <button
            key={tab.id}
            onClick={() => onChange(tab.id)}
            aria-current={active ? 'page' : undefined}
            className={`rounded-chip px-3 py-2 font-display text-[11px] font-semibold uppercase tracking-stamp transition-colors ${
              active ? 'bg-gold-400 text-ink' : 'text-muted hover:bg-line/40 hover:text-chalk'
            }`}
          >
            {tab.label}
          </button>
        );
      })}
    </nav>
  );
}
