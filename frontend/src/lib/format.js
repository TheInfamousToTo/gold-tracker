/**
 * Formats a number for display, falling back to a zero of the same
 * precision when the value is missing or unparseable — the API returns
 * null for anything that has no price data yet.
 */
export function fmt(value, decimals = 3) {
  const n = Number(value);
  if (!Number.isFinite(n)) return (0).toFixed(decimals);
  return n.toLocaleString(undefined, {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });
}

export function fmtDate(value) {
  if (!value) return '—';
  return new Date(value).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}
