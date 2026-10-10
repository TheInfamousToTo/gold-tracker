import { useState } from 'react';
import { Button } from '../ui/Button.jsx';
import { inputClass } from '../ui/Field.jsx';

const MAX_PLAN = 500;

/**
 * "I'm about to do X today — good idea?" The answer arrives as a review
 * card in the list below, through the same polling as Analyse.
 */
export function ReviewBox({ disabled, busy, onReview }) {
  const [plan, setPlan] = useState('');

  const submit = async (e) => {
    e.preventDefault();
    if (await onReview(plan.trim())) setPlan('');
  };

  return (
    <form onSubmit={submit} className="mb-5 space-y-2">
      <label htmlFor="plan" className="stamp block">
        Review my plan
      </label>
      <textarea
        id="plan"
        rows="2"
        maxLength={MAX_PLAN}
        placeholder="e.g. Instead of a game top-up I'll buy 15 BHD of gold today"
        value={plan}
        onChange={(e) => setPlan(e.target.value)}
        className={`${inputClass} resize-none`}
        disabled={disabled}
      />
      <div className="flex items-center justify-between gap-3">
        <span className="text-[11px] text-muted">
          Checked against prices, your budget and today's news. Takes a minute or two.
        </span>
        <Button type="submit" size="sm" variant="secondary" loading={busy} loadingLabel="Reviewing"
          disabled={disabled || plan.trim() === ''}>
          Review
        </Button>
      </div>
    </form>
  );
}
