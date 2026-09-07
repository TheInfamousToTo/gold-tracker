import { describe, it, expect } from 'vitest';
import { massByPurity } from './holdings.js';

const item = (over = {}) => ({
  metal_type: 'gold',
  purity_karat: 21,
  purity_fineness: null,
  weight_grams: 10,
  price_paid_total: 300,
  current_value: 320,
  ...over,
});

describe('massByPurity', () => {
  it('returns nothing for an empty portfolio', () => {
    expect(massByPurity([])).toEqual([]);
    expect(massByPurity(undefined)).toEqual([]);
  });

  it('sums mass, cost, and value across items of the same purity', () => {
    const [gold] = massByPurity([
      item({ weight_grams: 12.3, price_paid_total: 340, current_value: 360 }),
      item({ weight_grams: 30, price_paid_total: 840.4, current_value: 890.1 }),
    ]);

    expect(gold.rows).toHaveLength(1);
    expect(gold.rows[0].grams).toBeCloseTo(42.3, 6);
    expect(gold.rows[0].paid).toBeCloseTo(1180.4, 6);
    expect(gold.rows[0].value).toBeCloseTo(1250.1, 6);
  });

  it('reports the average entry price per gram, not the average of the rates', () => {
    // Two purchases at different rates: the blended entry is total
    // paid over total mass, which a mean of price_per_gram_paid would
    // get wrong whenever the masses differ.
    const [gold] = massByPurity([
      item({ weight_grams: 1, price_paid_total: 40 }),
      item({ weight_grams: 9, price_paid_total: 270 }),
    ]);

    expect(gold.rows[0].avgEntry).toBeCloseTo(31, 6);
  });

  it('splits rows by purity and keeps the highest purity first', () => {
    const [gold] = massByPurity([
      item({ purity_karat: 21 }),
      item({ purity_karat: 24 }),
      item({ purity_karat: 18 }),
    ]);

    expect(gold.rows.map((r) => r.purity)).toEqual([24, 21, 18]);
  });

  it('groups by metal, gold before silver', () => {
    const groups = massByPurity([
      item({ metal_type: 'silver', purity_karat: null, purity_fineness: 925 }),
      item({ metal_type: 'gold', purity_karat: 21 }),
    ]);

    expect(groups.map((g) => g.metal)).toEqual(['gold', 'silver']);
    expect(groups[1].rows[0].purity).toBe(925);
  });

  it('treats an item with no metal recorded as gold', () => {
    const groups = massByPurity([item({ metal_type: undefined })]);
    expect(groups.map((g) => g.metal)).toEqual(['gold']);
  });

  it('subtotals each metal', () => {
    const [gold] = massByPurity([
      item({ purity_karat: 24, weight_grams: 10, price_paid_total: 325, current_value: 340.2 }),
      item({ purity_karat: 21, weight_grams: 42.3, price_paid_total: 1180.4, current_value: 1250.1 }),
    ]);

    expect(gold.subtotal.grams).toBeCloseTo(52.3, 6);
    expect(gold.subtotal.paid).toBeCloseTo(1505.4, 6);
    expect(gold.subtotal.value).toBeCloseTo(1590.3, 6);
  });

  // current_value is null until a price has been recorded, and the
  // portfolio endpoint returns it that way. A group with no priced
  // item must not claim to be worth zero.
  it('reports a null value when nothing in the group is priced', () => {
    const [gold] = massByPurity([item({ current_value: null })]);

    expect(gold.rows[0].value).toBeNull();
    expect(gold.subtotal.value).toBeNull();
    expect(gold.rows[0].grams).toBe(10);
  });

  it('sums the priced items when only some of the group is priced', () => {
    const [gold] = massByPurity([
      item({ current_value: 320 }),
      item({ current_value: null }),
    ]);

    expect(gold.rows[0].value).toBeCloseTo(320, 6);
  });

  it('accepts the strings the API returns for numerics', () => {
    const [gold] = massByPurity([
      item({ purity_karat: '21', weight_grams: '10.5', price_paid_total: '300', current_value: '320' }),
    ]);

    expect(gold.rows[0].purity).toBe(21);
    expect(gold.rows[0].grams).toBeCloseTo(10.5, 6);
    expect(gold.rows[0].value).toBeCloseTo(320, 6);
  });

  it('leaves the average entry null for a weightless group', () => {
    const [gold] = massByPurity([item({ weight_grams: 0 })]);
    expect(gold.rows[0].avgEntry).toBeNull();
  });
});
