import { describe, it, expect } from 'vitest';
import { fineness, KARAT_OPTIONS } from './karat.js';

describe('fineness', () => {
  it('returns the standard trade marks', () => {
    expect(fineness(24)).toBe(999);
    expect(fineness(22)).toBe(916);
    expect(fineness(21)).toBe(875);
    expect(fineness(18)).toBe(750);
  });

  it('accepts string karats, as the API returns them', () => {
    expect(fineness('21')).toBe(875);
  });

  it('computes a mark for uncommon purities', () => {
    expect(fineness(14)).toBe(583);
  });

  it('returns a dash for missing or nonsense values', () => {
    expect(fineness(null)).toBe('—');
    expect(fineness(0)).toBe('—');
    expect(fineness('abc')).toBe('—');
  });
});

describe('KARAT_OPTIONS', () => {
  it('covers the four purities the form offers', () => {
    expect(KARAT_OPTIONS.map((o) => o.karat)).toEqual([24, 22, 21, 18]);
  });
});
