import { describe, it, expect } from 'vitest';
import { recordableBuy, purchaseDraft, confidencePct, verdictLabel, VERDICT_TONE } from './advice.js';

describe('recordableBuy', () => {
  const buy = { kind: 'analysis', signal_type: 'BUY', metal: 'gold', amount_bhd: 40, amount_grams: 0.79 };

  it('returns the buy of an analysis', () => {
    expect(recordableBuy(buy)).toEqual({ metal: 'gold', amountBhd: 40, grams: 0.79 });
  });

  it("uses a review's suggestion, not its verdict", () => {
    const review = { ...buy, kind: 'review', signal_type: 'ADJUST', suggested_action: 'BUY' };
    expect(recordableBuy(review)).not.toBeNull();
    expect(recordableBuy({ ...review, suggested_action: 'HOLD' })).toBeNull();
  });

  it('ignores sells, holds and buys without a size', () => {
    expect(recordableBuy({ ...buy, signal_type: 'SELL' })).toBeNull();
    expect(recordableBuy({ ...buy, amount_bhd: null })).toBeNull();
    expect(recordableBuy({ ...buy, metal: '' })).toBeNull();
  });
});

describe('purchaseDraft', () => {
  it('pre-fills an investment-grade purchase', () => {
    expect(purchaseDraft({ metal: 'silver', amountBhd: 10, grams: 13.5 }, '2026-10-10')).toMatchObject({
      purchase_date: '2026-10-10',
      metal_type: 'silver',
      purity: '999',
      weight_grams: '13.5',
      price_paid_total: '10',
    });
    expect(purchaseDraft({ metal: 'gold', amountBhd: 40, grams: 0.79 }).purity).toBe('24');
  });
});

describe('confidencePct', () => {
  it('rounds and bounds', () => {
    expect(confidencePct(0.666)).toBe(67);
    expect(confidencePct(1.4)).toBe(100);
    expect(confidencePct(null)).toBeNull();
  });
});

describe('verdicts', () => {
  it('labels and colours every verdict', () => {
    expect(verdictLabel('CUT_LOSS')).toBe('Cut loss');
    expect(verdictLabel('BUY')).toBe('BUY');
    expect(VERDICT_TONE.CUT_LOSS).toBe('bad');
    expect(VERDICT_TONE.HOLD).toBe('idle');
  });
});
