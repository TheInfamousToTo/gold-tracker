import { useState } from 'react';
import { apiRequest } from '../../api/client.js';
import { METALS } from '../../lib/purity.js';
import { METAL_RATE_FIELD } from '../../lib/prices.js';
import { Card } from '../ui/Card.jsx';
import { Button } from '../ui/Button.jsx';
import { Field, inputClass } from '../ui/Field.jsx';

const EMPTY = {
  metal: 'gold',
  price_date: new Date().toISOString().split('T')[0],
  rate: '',
};

/** What each metal's headline quote is, and what it derives. */
const RATE_FIELD_LABEL = {
  gold: { label: '24K price', hint: 'BHD per gram. 22K, 21K and 18K are derived.' },
  silver: { label: '999 price', hint: 'BHD per gram. 925 and 900 are derived.' },
};

/**
 * Manual spot entry. Prices normally arrive from the n8n feed; this is
 * for filling a gap by hand.
 */
export function PriceForm({ onSaved }) {
  const [form, setForm] = useState(EMPTY);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);

  const set = (key) => (e) => setForm((f) => ({ ...f, [key]: e.target.value }));
  const field = RATE_FIELD_LABEL[form.metal] || RATE_FIELD_LABEL.gold;

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      // The API reads only its own metal's rate column, so the value
      // is posted under the name that metal uses.
      await apiRequest('/api/prices', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          metal: form.metal,
          price_date: form.price_date,
          [METAL_RATE_FIELD[form.metal]]: Number(form.rate),
        }),
      });
      setForm({ ...EMPTY, metal: form.metal, price_date: form.price_date });
      onSaved();
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card title="Record spot price">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <Field label="Metal" htmlFor="price_metal">
            <select id="price_metal" value={form.metal} onChange={set('metal')} className={inputClass}>
              {METALS.map((m) => (
                <option key={m.value} value={m.value} className="bg-ink-raised">
                  {m.label}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Date" htmlFor="price_date">
            <input id="price_date" type="date" required value={form.price_date}
              onChange={set('price_date')} className={inputClass} />
          </Field>
        </div>
        <Field label={field.label} htmlFor="price_rate" hint={field.hint}>
          <input id="price_rate" type="number" step="0.001" min="0.001" required
            placeholder="0.000" value={form.rate}
            onChange={set('rate')} className={`${inputClass} font-mono`} />
        </Field>
        {error && (
          <p className="flex items-center gap-2 rounded-chip border border-bad/30 border-l-4 border-l-bad bg-bad/10 px-3 py-2 text-sm text-bad-bright">
            <span className="andon andon-bad">Error</span>
            {error}
          </p>
        )}
        <Button type="submit" variant="secondary" size="lg" loading={submitting} loadingLabel="Saving">
          Save price
        </Button>
      </form>
    </Card>
  );
}
