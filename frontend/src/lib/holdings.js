/**
 * Aggregation over the portfolio's item list.
 *
 * The holdings table answers "what do I own"; this answers "how much
 * metal do I hold at each purity", which is the number a dealer quotes
 * against. It is derived entirely from the rows /api/portfolio already
 * returns, so it costs no extra request.
 */

/** Metals in the order they are shown. Anything else sorts after. */
const METAL_ORDER = ['gold', 'silver'];

/**
 * The purity a row is grouped by: karat for gold, millesimal fineness
 * for silver. Which column carries it depends on the metal, so this is
 * the one place that branch lives.
 */
function purityOf(item, metal) {
  const raw = metal === 'silver' ? item.purity_fineness : item.purity_karat;
  const n = Number(raw);
  return Number.isFinite(n) ? n : null;
}

function num(value) {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

/**
 * Value is summed separately from mass and cost because the API returns
 * null for it until a price has been recorded. A group where nothing is
 * priced reports null rather than zero — a holding of unknown value and
 * a holding worth nothing are not the same claim.
 */
function addValue(acc, raw) {
  // Number(null) is 0, not NaN, so an unpriced row would otherwise
  // count as a real zero and turn the group's null into a figure.
  if (raw === null || raw === undefined || raw === '') return;
  const n = Number(raw);
  if (!Number.isFinite(n)) return;
  acc.value = (acc.value ?? 0) + n;
}

/**
 * Groups items by metal, then by purity within each metal.
 *
 * Returns one entry per metal — `{ metal, rows, subtotal }` — with rows
 * ordered by descending purity, since that is how a case is laid out.
 * `avgEntry` is total paid over total mass, the blended rate actually
 * paid, not the mean of the per-item rates.
 */
export function massByPurity(items) {
  if (!Array.isArray(items) || items.length === 0) return [];

  const metals = new Map();

  for (const item of items) {
    const metal = item.metal_type || 'gold';
    const purity = purityOf(item, metal);

    if (!metals.has(metal)) {
      metals.set(metal, new Map());
    }
    const rows = metals.get(metal);

    if (!rows.has(purity)) {
      rows.set(purity, { purity, grams: 0, paid: 0, value: null, avgEntry: null });
    }
    const row = rows.get(purity);

    row.grams += num(item.weight_grams);
    row.paid += num(item.price_paid_total);
    addValue(row, item.current_value);
  }

  return [...metals.entries()]
    .sort((a, b) => metalRank(a[0]) - metalRank(b[0]))
    .map(([metal, rowsByPurity]) => {
      const rows = [...rowsByPurity.values()].sort(byPurityDescending);
      for (const row of rows) {
        row.avgEntry = row.grams > 0 ? row.paid / row.grams : null;
      }
      return { metal, rows, subtotal: subtotalOf(rows) };
    });
}

function metalRank(metal) {
  const i = METAL_ORDER.indexOf(metal);
  return i === -1 ? METAL_ORDER.length : i;
}

/** Highest purity first; an unrecorded purity sorts last. */
function byPurityDescending(a, b) {
  if (a.purity === null) return 1;
  if (b.purity === null) return -1;
  return b.purity - a.purity;
}

function subtotalOf(rows) {
  const total = { grams: 0, paid: 0, value: null };
  for (const row of rows) {
    total.grams += row.grams;
    total.paid += row.paid;
    addValue(total, row.value);
  }
  return total;
}
