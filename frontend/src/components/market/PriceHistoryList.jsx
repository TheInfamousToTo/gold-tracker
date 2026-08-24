import { fmt, fmtDate } from '../../lib/format.js';
import { Card } from '../ui/Card.jsx';

export function PriceHistoryList({ prices }) {
  return (
    <Card title="Recent prices" padded={false}>
      {prices.length === 0 ? (
        <p className="px-5 py-8 text-center text-xs text-muted">Nothing recorded yet.</p>
      ) : (
        <ul className="max-h-80 divide-y divide-line overflow-y-auto">
          {prices.map((p) => (
            <li key={p.id} className="flex items-center justify-between px-5 py-2.5">
              <span className="text-xs text-muted">{fmtDate(p.price_date)}</span>
              <span className="font-mono text-sm text-chalk">{fmt(p.price_per_gram_24k, 3)}</span>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
