import { describe, it, expect } from 'vitest';
import {
  fineness,
  purityMark,
  purityColumns,
  purityOptions,
  KARAT_OPTIONS,
  SILVER_OPTIONS,
  METALS,
} from './purity.js';

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

describe('SILVER_OPTIONS', () => {
  it('covers the three marks silver is sold in', () => {
    expect(SILVER_OPTIONS.map((o) => o.fineness)).toEqual([999, 925, 900]);
  });
});

describe('METALS', () => {
  it('offers gold and silver', () => {
    expect(METALS.map((m) => m.value)).toEqual(['gold', 'silver']);
  });
});

describe('purityOptions', () => {
  it('gives karats for gold and fineness marks for silver', () => {
    expect(purityOptions('gold')).toBe(KARAT_OPTIONS);
    expect(purityOptions('silver')).toBe(SILVER_OPTIONS);
  });

  it('falls back to gold when the metal is missing', () => {
    expect(purityOptions(undefined)).toBe(KARAT_OPTIONS);
  });
});

describe('purityMark', () => {
  it('marks gold with its millesimal fineness and karat', () => {
    const mark = purityMark('gold', { karat: 21 });
    expect(mark.mark).toBe(875);
    expect(mark.qualifier).toBe('21K');
    expect(mark.title).toBe('Gulf standard');
  });

  // Silver has no karat notation, so the qualifier carries the trade
  // name a buyer would actually say instead.
  it('marks silver with its fineness and trade name', () => {
    const mark = purityMark('silver', { fineness: 925 });
    expect(mark.mark).toBe(925);
    expect(mark.qualifier).toBe('Sterling');
    expect(mark.title).toBe('Jewellery and flatware');
  });

  it('reads the silver fineness, not the karat column', () => {
    // A silver row leaves purity_karat null; reading it would render
    // every silver piece as a dash.
    const mark = purityMark('silver', { karat: null, fineness: 999 });
    expect(mark.mark).toBe(999);
  });

  it('handles an unlisted silver fineness', () => {
    const mark = purityMark('silver', { fineness: 800 });
    expect(mark.mark).toBe(800);
    expect(mark.qualifier).toBe('Silver');
  });

  it('returns a dash rather than throwing on missing purity', () => {
    expect(purityMark('silver', {}).mark).toBe('—');
    expect(purityMark('gold', {}).mark).toBe('—');
    expect(purityMark('gold').mark).toBe('—');
  });
});

describe('purityColumns', () => {
  it('puts a gold purity in the karat column only', () => {
    expect(purityColumns('gold', '21')).toEqual({ purity_karat: 21, purity_fineness: null });
  });

  // Sending both columns is rejected by the schema's CHECK constraint,
  // so the form must never fill the one its metal does not use.
  it('puts a silver purity in the fineness column only', () => {
    expect(purityColumns('silver', '925')).toEqual({ purity_karat: null, purity_fineness: 925 });
  });

  it('nulls both for a purity that is missing or nonsense', () => {
    expect(purityColumns('gold', '')).toEqual({ purity_karat: null, purity_fineness: null });
    expect(purityColumns('gold', 0)).toEqual({ purity_karat: null, purity_fineness: null });
    expect(purityColumns('silver', 'abc')).toEqual({ purity_karat: null, purity_fineness: null });
  });
});
