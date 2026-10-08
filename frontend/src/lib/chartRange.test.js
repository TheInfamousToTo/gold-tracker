import { describe, it, expect } from 'vitest';
import { rangeStart, sliceRange, availableRanges, rangeStats } from './chartRange.js';

const series = (dates) => dates.map((date, i) => ({ date, price: 10 + i }));
const keys = (ranges) => ranges.map((r) => r.key);

describe('rangeStart', () => {
  it('steps back from the newest day', () => {
    expect(rangeStart('5D', '2026-10-08')).toBe('2026-10-03');
    expect(rangeStart('1M', '2026-10-08')).toBe('2026-09-08');
    expect(rangeStart('1Y', '2026-10-08')).toBe('2025-10-08');
    expect(rangeStart('2Y', '2026-10-08')).toBe('2024-10-08');
    expect(rangeStart('5Y', '2026-10-08')).toBe('2021-10-08');
    expect(rangeStart('10Y', '2026-10-08')).toBe('2016-10-08');
  });

  it('crosses a year boundary', () => {
    expect(rangeStart('3M', '2026-01-15')).toBe('2025-10-15');
  });

  // Without the clamp, a month back from 31 March overflows to 3 March
  // and "1M" quietly becomes a four-day window.
  it('clamps to the end of a shorter month', () => {
    expect(rangeStart('1M', '2026-03-31')).toBe('2026-02-28');
    expect(rangeStart('1Y', '2028-02-29')).toBe('2027-02-28');
  });

  it('starts YTD on 1 January', () => {
    expect(rangeStart('YTD', '2026-10-08')).toBe('2026-01-01');
  });

  it('has no start for All', () => {
    expect(rangeStart('ALL', '2026-10-08')).toBeNull();
  });
});

describe('sliceRange', () => {
  it('keeps the points inside the window', () => {
    const data = series(['2026-09-01', '2026-10-01', '2026-10-05', '2026-10-08']);
    expect(sliceRange(data, '5D').map((d) => d.date)).toEqual(['2026-10-05', '2026-10-08']);
    expect(sliceRange(data, 'ALL')).toHaveLength(4);
  });
});

describe('availableRanges', () => {
  it('offers only All for a single point', () => {
    expect(keys(availableRanges(series(['2026-10-08'])))).toEqual(['ALL']);
  });

  // Two months of history: 1M is a real narrowing, 3M and up would
  // just repeat All.
  it('hides windows that reach past the first day', () => {
    const data = series(['2026-08-08', '2026-09-01', '2026-09-20', '2026-10-05', '2026-10-08']);
    expect(keys(availableRanges(data))).toEqual(['5D', '1M', 'ALL']);
  });

  it('hides windows with fewer than two points', () => {
    const data = series(['2026-08-08', '2026-10-08']);
    expect(keys(availableRanges(data))).not.toContain('5D');
  });

  // Daily rows for three years: everything up to 2Y can be filled, 5Y
  // and 10Y would only repeat All.
  it('offers the multi-year windows once the history reaches them', () => {
    const days = [];
    for (let t = Date.UTC(2023, 9, 1); t <= Date.UTC(2026, 9, 8); t += 864e5) {
      days.push(new Date(t).toISOString().slice(0, 10));
    }
    expect(keys(availableRanges(series(days)))).toEqual(
      ['5D', '1M', '3M', '6M', 'YTD', '1Y', '2Y', 'ALL'],
    );
  });
});

describe('rangeStats', () => {
  it('measures change from the first point to the last', () => {
    const s = rangeStats([{ price: 20 }, { price: 15 }, { price: 25 }]);
    expect(s.change).toBe(5);
    expect(s.pct).toBe(25);
    expect(s.low).toBe(15);
    expect(s.high).toBe(25);
  });

  it('returns null for an empty window', () => {
    expect(rangeStats([])).toBeNull();
  });

});
