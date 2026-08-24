import { fineness, KARAT_LABEL } from '../../lib/karat.js';

/**
 * Purity shown the way it is actually stamped on the metal: the
 * millesimal fineness mark. 21K gold in the Gulf is marked 875, not
 * "21K Standard", so the chip carries the real information a buyer
 * reads off the piece. Purity is a fact, not a status, so it is set in
 * neutral type — a higher karat is not a "better" row.
 */
export function Hallmark({ karat, size = 'md' }) {
  const mark = fineness(karat);
  const sizes = {
    sm: 'text-[10px] px-1.5 py-0.5',
    md: 'text-xs px-2 py-1',
  };

  return (
    <span
      title={KARAT_LABEL[karat] || `${karat}K`}
      className={`inline-flex items-center gap-1.5 rounded-chip border border-line-bright bg-ink-sunken font-mono font-semibold text-chalk ${sizes[size]}`}
    >
      {mark}
      <span className="text-[0.8em] font-normal text-muted">{karat}K</span>
    </span>
  );
}
