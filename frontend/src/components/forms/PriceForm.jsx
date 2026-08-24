import { useState } from 'react';
import { apiRequest } from '../../api/client.js';
import { Card } from '../ui/Card.jsx';
import { Button } from '../ui/Button.jsx';
import { Field, inputClass } from '../ui/Field.jsx';

const EMPTY = {
  price_date: new Date().toISOString().split('T')[0],
  price_per_gram_24k: '',
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

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await apiRequest('/api/prices', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      setForm({ ...EMPTY, price_date: form.price_date });
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
        <Field label="Date" htmlFor="price_date">
          <input id="price_date" type="date" required value={form.price_date}
            onChange={set('price_date')} className={inputClass} />
        </Field>
        <Field label="24K price" htmlFor="price_per_gram_24k" hint="BHD per gram. 22K, 21K and 18K are derived.">
          <input id="price_per_gram_24k" type="number" step="0.001" min="0.001" required
            placeholder="0.000" value={form.price_per_gram_24k}
            onChange={set('price_per_gram_24k')} className={`${inputClass} font-mono`} />
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
