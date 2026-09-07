import { purityMark } from '../../lib/purity.js';

/**
 * Purity shown the way it is actually stamped on the metal: the
 * millesimal fineness mark. 21K gold in the Gulf is marked 875, not
 * "21K Standard", so the chip carries the real information a buyer
 * reads off the piece. Silver has no karat notation at all — 925 is
 * sterling — so the qualifier carries the trade name there instead.
 *
 * Purity is a fact, not a status, so it is set in neutral type — a
 * higher karat is not a "better" row. Metal is distinguished by the
 * mark's own tone rather than a status colour: gold reads warm, silver
 * reads cool, and neither means good or bad.
 */
const METAL_TONE = {
  gold: 'text-chalk',
  silver: 'text-chalk/85',
};

export function Hallmark({ metal = 'gold', karat, fineness, size = 'md' }) {
  const { mark, qualifier, title } = purityMark(metal, { karat, fineness });
  const sizes = {
    sm: 'text-[10px] px-1.5 py-0.5',
    md: 'text-xs px-2 py-1',
  };

  return (
    <span
      title={title}
      className={`inline-flex items-center gap-1.5 rounded-chip border border-line-bright bg-ink-sunken font-mono font-semibold ${
        METAL_TONE[metal] || METAL_TONE.gold
      } ${sizes[size]}`}
    >
      {mark}
      {qualifier && <span className="text-[0.8em] font-normal text-muted">{qualifier}</span>}
    </span>
  );
}
