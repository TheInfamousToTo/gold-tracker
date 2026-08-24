import { describe, it, expect } from 'vitest';
import { fmt, fmtDate } from './format.js';

describe('fmt', () => {
  it('formats a number with the default 3 decimals', () => {
    expect(fmt(45.1)).toBe('45.100');
  });

  it('formats with a custom decimal count', () => {
    expect(fmt(45.126, 2)).toBe('45.13');
  });

  it('returns a zero string for non-finite input', () => {
    expect(fmt(undefined)).toBe('0.000');
    expect(fmt(null)).toBe('0.000');
    expect(fmt('not a number')).toBe('0.000');
  });

  // The original inline helper always returned '0.000', so a 2-decimal
  // column rendered a 3-decimal fallback.
  it('respects the decimal count in the fallback', () => {
    expect(fmt(undefined, 2)).toBe('0.00');
  });

  it('handles zero and negatives', () => {
    expect(fmt(0, 2)).toBe('0.00');
    expect(fmt(-12.5, 2)).toBe('-12.50');
  });
});

describe('fmtDate', () => {
  it('returns an em dash for empty input', () => {
    expect(fmtDate(null)).toBe('—');
    expect(fmtDate('')).toBe('—');
    expect(fmtDate(undefined)).toBe('—');
  });

  it('formats a date string', () => {
    const result = fmtDate('2026-08-01');
    expect(result).toMatch(/2026/);
    expect(result).toMatch(/Aug/);
  });
});
