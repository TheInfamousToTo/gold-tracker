import { METALS } from '../../lib/purity.js';

/**
 * Which metal the market view is showing. Like the nav tabs, this is a
 * neutral choice of view rather than a status, so the selected metal is
 * marked by contrast alone and carries no colour.
 */
export function MetalToggle({ metal, onChange }) {
  return (
    <div className="flex gap-1" role="group" aria-label="Metal">
      {METALS.map((m) => {
        const active = metal === m.value;
        return (
          <button
            key={m.value}
            type="button"
            onClick={() => onChange(m.value)}
            aria-pressed={active}
            className={`rounded-chip px-2.5 py-1 font-display text-[10px] font-semibold uppercase tracking-stamp transition-colors ${
              active ? 'bg-chalk text-ink' : 'text-muted hover:bg-line/40 hover:text-chalk'
            }`}
          >
            {m.label}
          </button>
        );
      })}
    </div>
  );
}
