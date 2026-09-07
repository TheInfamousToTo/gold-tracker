/**
 * Purity, in the notations each metal is actually traded and stamped
 * in: gold by karat, silver by millesimal fineness alone. Both end up
 * as a millesimal mark on the piece, which is what the UI shows.
 */

export const METALS = [
  { value: 'gold', label: 'Gold' },
  { value: 'silver', label: 'Silver' },
];

export const KARAT_OPTIONS = [
  { value: '24', karat: 24, fineness: 999, label: '24K — 999', description: 'Investment grade' },
  { value: '22', karat: 22, fineness: 916, label: '22K — 916', description: 'Traditional jewellery' },
  { value: '21', karat: 21, fineness: 875, label: '21K — 875', description: 'Gulf standard' },
  { value: '18', karat: 18, fineness: 750, label: '18K — 750', description: 'Fine jewellery' },
];

/**
 * Silver has no karat notation — sterling is 925, full stop — so these
 * carry the trade name where gold carries its karat.
 */
export const SILVER_OPTIONS = [
  { value: '999', fineness: 999, short: 'Fine', label: '999 — Fine', description: 'Investment grade' },
  { value: '925', fineness: 925, short: 'Sterling', label: '925 — Sterling', description: 'Jewellery and flatware' },
  { value: '900', fineness: 900, short: 'Coin', label: '900 — Coin', description: 'Coin silver' },
];

export const KARAT_LABEL = Object.fromEntries(
  KARAT_OPTIONS.map((o) => [o.karat, o.description]),
);

const SILVER_BY_FINENESS = Object.fromEntries(
  SILVER_OPTIONS.map((o) => [o.fineness, o]),
);

const FINENESS = Object.fromEntries(KARAT_OPTIONS.map((o) => [o.karat, o.fineness]));

/** The purity choices a metal offers, for the item and price forms. */
export function purityOptions(metal) {
  return metal === 'silver' ? SILVER_OPTIONS : KARAT_OPTIONS;
}

/**
 * The millesimal mark for a karat value. Falls back to computing it for
 * purities outside the four common marks, since the API accepts any.
 */
export function fineness(karat) {
  const k = Number(karat);
  if (FINENESS[k]) return FINENESS[k];
  if (!Number.isFinite(k) || k <= 0) return '—';
  return Math.round((k / 24) * 1000);
}

/**
 * How a holding's purity is written on the piece.
 *
 * `mark` is the millesimal number stamped on the metal; `qualifier` is
 * the short form a buyer would say out loud — a karat for gold, the
 * trade name for silver. `title` is the longer description for a
 * tooltip.
 */
export function purityMark(metal, { karat, fineness: finenessValue } = {}) {
  if (metal === 'silver') {
    const f = Number(finenessValue);
    if (!Number.isFinite(f) || f <= 0) return { mark: '—', qualifier: '', title: 'Silver' };
    const known = SILVER_BY_FINENESS[f];
    return {
      mark: f,
      qualifier: known ? known.short : 'Silver',
      title: known ? known.description : `${f} silver`,
    };
  }

  const k = Number(karat);
  return {
    mark: fineness(karat),
    qualifier: Number.isFinite(k) && k > 0 ? `${k}K` : '',
    title: KARAT_LABEL[k] || (Number.isFinite(k) && k > 0 ? `${k}K gold` : 'Gold'),
  };
}
