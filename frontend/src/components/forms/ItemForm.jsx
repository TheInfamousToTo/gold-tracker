import { useState, useEffect } from 'react';
import { apiRequest } from '../../api/client.js';
import { KARAT_OPTIONS } from '../../lib/karat.js';
import { Card } from '../ui/Card.jsx';
import { Button } from '../ui/Button.jsx';
import { Field, inputClass } from '../ui/Field.jsx';

const EMPTY = {
  purchase_date: new Date().toISOString().split('T')[0],
  item_name: '',
  metal_type: 'gold',
  purity_karat: '21',
  weight_grams: '',
  price_paid_total: '',
  vendor: '',
  notes: '',
};

function toFormState(item) {
  if (!item) return EMPTY;
  return {
    purchase_date: item.purchase_date,
    item_name: item.item_name,
    metal_type: 'gold',
    purity_karat: String(item.purity_karat),
    weight_grams: String(item.weight_grams),
    price_paid_total: String(item.price_paid_total),
    vendor: item.vendor || '',
    notes: item.notes || '',
  };
}

export function ItemForm({ editingItem, onSaved, onCancelEdit }) {
  const [form, setForm] = useState(() => toFormState(editingItem));
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    setForm(toFormState(editingItem));
    setError(null);
  }, [editingItem]);

  const set = (key) => (e) => setForm((f) => ({ ...f, [key]: e.target.value }));

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await apiRequest(editingItem ? `/api/items/${editingItem.id}` : '/api/items', {
        method: editingItem ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      setForm(EMPTY);
      onSaved();
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card
      title={editingItem ? 'Edit purchase' : 'Add purchase'}
      actions={
        editingItem && (
          <Button variant="ghost" size="sm" onClick={onCancelEdit}>
            Cancel
          </Button>
        )
      }
    >
      <form onSubmit={handleSubmit} className="space-y-5">
        <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
          <Field label="Purchase date" htmlFor="purchase_date">
            <input id="purchase_date" type="date" required value={form.purchase_date}
              onChange={set('purchase_date')} className={inputClass} />
          </Field>
          <Field label="Item" htmlFor="item_name">
            <input id="item_name" type="text" required placeholder="Swiss 10g bar"
              value={form.item_name} onChange={set('item_name')} className={inputClass} />
          </Field>
        </div>

        <div className="grid grid-cols-1 gap-5 md:grid-cols-3">
          <Field label="Purity" htmlFor="purity_karat">
            <select id="purity_karat" value={form.purity_karat} onChange={set('purity_karat')}
              className={inputClass}>
              {KARAT_OPTIONS.map((o) => (
                <option key={o.value} value={o.value} className="bg-ink-raised">
                  {o.label} · {o.description}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Mass" htmlFor="weight_grams" hint="Grams">
            <input id="weight_grams" type="number" step="0.001" min="0.001" required
              placeholder="0.000" value={form.weight_grams} onChange={set('weight_grams')}
              className={`${inputClass} font-mono`} />
          </Field>
          <Field label="Total paid" htmlFor="price_paid_total" hint="BHD, including making charges">
            <input id="price_paid_total" type="number" step="0.001" min="0" required
              placeholder="0.000" value={form.price_paid_total} onChange={set('price_paid_total')}
              className={`${inputClass} font-mono`} />
          </Field>
        </div>

        <Field label="Vendor" htmlFor="vendor">
          <input id="vendor" type="text" placeholder="Where you bought it" value={form.vendor}
            onChange={set('vendor')} className={inputClass} />
        </Field>

        <Field label="Notes" htmlFor="notes">
          <textarea id="notes" rows="2" placeholder="Certificate number, identifying marks"
            value={form.notes} onChange={set('notes')} className={`${inputClass} resize-none`} />
        </Field>

        {error && (
          <p className="flex items-center gap-2 rounded-chip border border-bad/30 border-l-4 border-l-bad bg-bad/10 px-3 py-2 text-sm text-bad-bright">
            <span className="andon andon-bad">Error</span>
            {error}
          </p>
        )}

        <Button type="submit" size="lg" loading={submitting}
          loadingLabel={editingItem ? 'Saving' : 'Adding'}>
          {editingItem ? 'Save changes' : 'Add purchase'}
        </Button>
      </form>
    </Card>
  );
}
