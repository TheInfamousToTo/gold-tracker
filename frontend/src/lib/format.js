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

/**
 * Whole days between a date and today, or null when there is no date.
 * Used to decide whether the price board is showing live data or a
 * stale figure — a number nobody has refreshed is the failure mode this
 * app is most likely to hide.
 */
export function daysSince(value) {
  if (!value) return null;
  const then = new Date(value);
  if (Number.isNaN(then.getTime())) return null;
  const dayMs = 24 * 60 * 60 * 1000;
  const midnight = (d) => Date.UTC(d.getFullYear(), d.getMonth(), d.getDate());
  return Math.max(0, Math.round((midnight(new Date()) - midnight(then)) / dayMs));
}
