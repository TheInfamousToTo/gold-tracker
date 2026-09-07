import { describe, it, expect } from 'vitest';
import { withRates, spotOf, METAL_FINE_LABEL } from './prices.js';

describe('withRates', () => {
  it('reads the 24K column for gold', () => {
    const [row] = withRates([{ price_date: '2026-08-01', price_per_gram_24k: 32.5 }], 'gold');
    expect(row.rate).toBe(32.5);
  });

  // Reading price_per_gram_24k off a silver row yields undefined, and
  // the whole chart silently empties. This is the guard for that.
  it('reads the 999 column for silver', () => {
    const [row] = withRates([{ price_date: '2026-08-01', price_per_gram_999: 0.412 }], 'silver');
    expect(row.rate).toBe(0.412);
  });

  it('keeps the original row fields', () => {
    const [row] = withRates([{ id: 7, price_date: '2026-08-01', price_per_gram_24k: 32.5 }], 'gold');
    expect(row.id).toBe(7);
    expect(row.price_date).toBe('2026-08-01');
  });

  it('falls back to gold for an unknown metal', () => {
    const [row] = withRates([{ price_per_gram_24k: 32.5 }], undefined);
    expect(row.rate).toBe(32.5);
  });

  it('returns an empty list for missing data', () => {
    expect(withRates(undefined, 'gold')).toEqual([]);
    expect(withRates(null, 'silver')).toEqual([]);
  });
});

describe('spotOf', () => {
  // The API returns newest first, so the spot is the head of the list.
  it('takes the newest quote', () => {
    const prices = withRates(
      [
        { price_date: '2026-08-02', price_per_gram_24k: 33 },
        { price_date: '2026-08-01', price_per_gram_24k: 32 },
      ],
      'gold',
    );
    expect(spotOf(prices)).toBe(33);
  });

  it('is undefined with no prices', () => {
    expect(spotOf([])).toBeUndefined();
    expect(spotOf(undefined)).toBeUndefined();
  });
});

describe('METAL_FINE_LABEL', () => {
  it('names each metal by the purity it is quoted at', () => {
    expect(METAL_FINE_LABEL.gold).toBe('24K');
    expect(METAL_FINE_LABEL.silver).toBe('999');
  });
});
