/**
 * Gold purity, in the two notations the trade uses: karat and the
 * millesimal fineness mark actually stamped on the piece.
 */
export const KARAT_OPTIONS = [
  { value: '24', karat: 24, fineness: 999, label: '24K — 999', description: 'Investment grade' },
  { value: '22', karat: 22, fineness: 916, label: '22K — 916', description: 'Traditional jewellery' },
  { value: '21', karat: 21, fineness: 875, label: '21K — 875', description: 'Gulf standard' },
  { value: '18', karat: 18, fineness: 750, label: '18K — 750', description: 'Fine jewellery' },
];

export const KARAT_LABEL = Object.fromEntries(
  KARAT_OPTIONS.map((o) => [o.karat, o.description]),
);

const FINENESS = Object.fromEntries(KARAT_OPTIONS.map((o) => [o.karat, o.fineness]));

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
