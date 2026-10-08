/**
 * Time windows for the price chart. Each window is anchored on the
 * newest recorded day rather than on today, so a feed that has gone
 * quiet for a week still shows a full "1M" instead of three weeks.
 */

export const RANGES = [
  { key: '5D', label: '5D', caption: 'Past 5 days' },
  { key: '1M', label: '1M', caption: 'Past month' },
  { key: '3M', label: '3M', caption: 'Past 3 months' },
  { key: '6M', label: '6M', caption: 'Past 6 months' },
  { key: 'YTD', label: 'YTD', caption: 'Year to date' },
  { key: '1Y', label: '1Y', caption: 'Past year' },
  { key: '2Y', label: '2Y', caption: 'Past 2 years' },
  { key: '5Y', label: '5Y', caption: 'Past 5 years' },
  { key: '10Y', label: '10Y', caption: 'Past 10 years' },
  { key: 'ALL', label: 'All', caption: 'All recorded' },
];

/**
 * Works on "YYYY-MM-DD" strings in UTC so no timezone shifts a day.
 * Month and year steps clamp to the target month's last day: a month
 * back from 31 March is 28 February, not 3 March.
 */
function shift(day, { days = 0, months = 0, years = 0 }) {
  const d = new Date(`${day}T00:00:00Z`);
  const totalMonths = d.getUTCMonth() + months + years * 12;
  const year = d.getUTCFullYear() + Math.floor(totalMonths / 12);
  const month = ((totalMonths % 12) + 12) % 12;
  const lastOfMonth = new Date(Date.UTC(year, month + 1, 0)).getUTCDate();
  const out = new Date(Date.UTC(year, month, Math.min(d.getUTCDate(), lastOfMonth) + days));
  return out.toISOString().slice(0, 10);
}

/** First day (inclusive) of a window ending on `last`, or null for all. */
export function rangeStart(key, last) {
  switch (key) {
    case '5D': return shift(last, { days: -5 });
    case '1M': return shift(last, { months: -1 });
    case '3M': return shift(last, { months: -3 });
    case '6M': return shift(last, { months: -6 });
    case 'YTD': return `${last.slice(0, 4)}-01-01`;
    case '1Y': return shift(last, { years: -1 });
    case '2Y': return shift(last, { years: -2 });
    case '5Y': return shift(last, { years: -5 });
    case '10Y': return shift(last, { years: -10 });
    default: return null;
  }
}

/** The points of an ascending `{ date }` series that fall in a window. */
export function sliceRange(data, key) {
  if (data.length === 0) return data;
  const start = rangeStart(key, data[data.length - 1].date);
  return start ? data.filter((d) => d.date >= start) : data;
}

/**
 * Windows worth offering for this series: one needs at least two points
 * to draw a line, and one that already reaches back past the first
 * recorded day would only repeat "All" under another name.
 */
export function availableRanges(data) {
  if (data.length < 2) return RANGES.filter((r) => r.key === 'ALL');
  const first = data[0].date;
  const last = data[data.length - 1].date;
  return RANGES.filter((r) => {
    const start = rangeStart(r.key, last);
    if (!start) return true;
    return start > first && data.filter((d) => d.date >= start).length >= 2;
  });
}

/** Change, low and high across a window's points. */
export function rangeStats(points) {
  if (points.length === 0) return null;
  const prices = points.map((p) => Number(p.price));
  const open = prices[0];
  const close = prices[prices.length - 1];
  const change = close - open;
  return {
    open,
    close,
    change,
    pct: open ? (change / open) * 100 : 0,
    low: Math.min(...prices),
    high: Math.max(...prices),
  };
}
