import { useEffect, useState } from 'react';
import { apiRequest } from '../../api/client.js';
import { Card } from '../ui/Card.jsx';
import { Button } from '../ui/Button.jsx';
import { Field, inputClass } from '../ui/Field.jsx';

const NUMBER_FIELDS = [
  { key: 'monthly_budget_bhd', label: 'Monthly budget', hint: 'BHD you put into metals each month', step: '0.5' },
  { key: 'reserve_bhd', label: 'Reserve', hint: 'BHD set aside for deep dips', step: '0.5' },
  { key: 'target_gold_pct', label: 'Gold share', hint: '% of the portfolio; silver is the rest', step: '1' },
  { key: 'stop_loss_pct', label: 'Stop-loss', hint: '% below what you paid, after the spread', step: '0.5' },
  { key: 'spread_pct_gold', label: 'Gold spread', hint: '% each way vs the 24K rate', step: '0.05' },
  { key: 'spread_pct_silver', label: 'Silver spread', hint: '% each way vs the 999 rate', step: '0.05' },
  { key: 'min_fee_bhd', label: 'Minimum fee', hint: 'BHD per buy; 0 if none', step: '0.1' },
];

/**
 * What the advisor sizes against. The reserve is edited by hand: the
 * app cannot see the owner's bank balance, and guessing it would be
 * worse than asking.
 */
export function AdvisorSettings() {
  const [form, setForm] = useState(null);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState(null);

  useEffect(() => {
    apiRequest('/api/ai/settings')
      .then((s) => setForm(s))
      .catch((err) => setMessage({ kind: 'error', text: err.message }));
  }, []);

  if (!form) {
    return (
      <Card title="Advisor settings">
        <p className="text-xs text-muted">{message?.text || 'Loading…'}</p>
      </Card>
    );
  }

  const set = (key) => (e) => setForm((f) => ({ ...f, [key]: e.target.value }));

  const save = async (e) => {
    e.preventDefault();
    setSaving(true);
    setMessage(null);
    try {
      const payload = { ...form };
      NUMBER_FIELDS.forEach(({ key }) => { payload[key] = Number(form[key]); });
      setForm(await apiRequest('/api/ai/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      }));
      setMessage({ kind: 'ok', text: 'Saved. The next analysis uses these.' });
    } catch (err) {
      setMessage({ kind: 'error', text: err.message });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card title="Advisor settings">
      <form onSubmit={save} className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          {NUMBER_FIELDS.map(({ key, label, hint, step }) => (
            <Field key={key} label={label} htmlFor={key} hint={hint}>
              <input id={key} type="number" min="0" step={step} required value={form[key]}
                onChange={set(key)} className={`${inputClass} font-mono`} />
            </Field>
          ))}
        </div>
        <label className="flex items-center gap-2 text-xs text-chalk">
          <input type="checkbox" checked={!!form.news_enabled}
            onChange={(e) => setForm((f) => ({ ...f, news_enabled: e.target.checked }))} />
          Check the news before each call (slower, uses more of the subscription)
        </label>
        {message && (
          <p className={`text-xs ${message.kind === 'error' ? 'text-bad-bright' : 'text-ok-bright'}`}>{message.text}</p>
        )}
        <Button type="submit" size="lg" loading={saving} loadingLabel="Saving">Save settings</Button>
      </form>
    </Card>
  );
}
