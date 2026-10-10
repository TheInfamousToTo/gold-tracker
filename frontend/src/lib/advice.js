/**
 * How each verdict is coloured. Buying is the normal state for a saver,
 * so it is green; a trim is amber — worth attention, not a failure —
 * and cutting a loss is the one red call. Hold is deliberately
 * colourless: if every card were coloured, none would signal anything.
 */
export const VERDICT_TONE = {
  BUY: 'ok',
  HOLD: 'idle',
  SELL: 'warn',
  CUT_LOSS: 'bad',
  GOOD_IDEA: 'ok',
  ADJUST: 'warn',
  BAD_IDEA: 'bad',
};

const LABELS = {
  CUT_LOSS: 'Cut loss',
  GOOD_IDEA: 'Good idea',
  BAD_IDEA: 'Bad idea',
};

/** A verdict as it should read on a badge. */
export function verdictLabel(type) {
  return LABELS[type] || type || '—';
}

/**
 * The action a card's "I did it" button records: the review's own
 * suggestion, or the analysis verdict itself. Only buys are recorded —
 * the purchase form adds holdings, and a sale is not a purchase.
 */
export function recordableBuy(signal) {
  const action = signal.kind === 'review' ? signal.suggested_action : signal.signal_type;
  if (action !== 'BUY') return null;
  if (!(signal.amount_bhd > 0) || !(signal.amount_grams > 0)) return null;
  if (signal.metal !== 'gold' && signal.metal !== 'silver') return null;
  return { metal: signal.metal, amountBhd: signal.amount_bhd, grams: signal.amount_grams };
}

/**
 * Pre-fills the purchase form from a recommended buy. Online buys are
 * investment grade, so the purity is 24K or 999; the owner corrects the
 * grams and price to what the bank or dealer actually filled.
 */
export function purchaseDraft(buy, today = new Date().toISOString().split('T')[0]) {
  const silver = buy.metal === 'silver';
  return {
    purchase_date: today,
    item_name: silver ? 'Online silver buy' : 'Online gold buy',
    metal_type: buy.metal,
    purity: silver ? '999' : '24',
    weight_grams: String(buy.grams),
    price_paid_total: String(buy.amountBhd),
    vendor: '',
    notes: 'Recorded from an advisor recommendation',
  };
}

/** Confidence as a whole percentage, or null when absent. */
export function confidencePct(value) {
  const n = Number(value);
  if (value == null || !Number.isFinite(n)) return null;
  return Math.round(Math.min(1, Math.max(0, n)) * 100);
}
