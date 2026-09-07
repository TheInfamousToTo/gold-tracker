/**
 * Gold and silver are quoted from separate tables with different rate
 * columns — price_per_gram_24k against price_per_gram_999. Rather than
 * making every chart, list, and masthead branch on the metal, the
 * series are normalised once here to a common `rate` field.
 */

export const METAL_RATE_FIELD = {
  gold: 'price_per_gram_24k',
  silver: 'price_per_gram_999',
};

/** How the metal's headline purity is written on the board. */
export const METAL_FINE_LABEL = {
  gold: '24K',
  silver: '999',
};

/**
 * Adds a `rate` field carrying whichever per-gram column this metal
 * uses, leaving the original row intact for anything that wants the
 * raw shape.
 */
export function withRates(prices, metal) {
  const field = METAL_RATE_FIELD[metal] || METAL_RATE_FIELD.gold;
  if (!Array.isArray(prices)) return [];
  return prices.map((p) => ({ ...p, rate: Number(p[field]) }));
}

/** The most recent quote, or undefined when the series is empty. */
export function spotOf(prices) {
  return prices?.[0]?.rate;
}
