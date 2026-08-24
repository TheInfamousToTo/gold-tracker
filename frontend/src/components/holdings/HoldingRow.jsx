import { fmt, fmtDate } from '../../lib/format.js';
import { Hallmark } from '../ui/Hallmark.jsx';

export function HoldingRow({ item, onEdit, onDelete }) {
  const gain = item.gain_loss ?? 0;
  const tone = gain >= 0 ? 'text-patina' : 'text-oxide';

  return (
    <tr className="border-t border-line transition-colors hover:bg-line/20">
      <td className="px-5 py-4">
        <p className="text-sm font-medium text-chalk">{item.item_name}</p>
        <p className="mt-0.5 text-xs text-muted">
          {item.vendor || 'Source not recorded'} · {fmtDate(item.purchase_date)}
        </p>
      </td>
      <td className="px-5 py-4">
        <Hallmark karat={item.purity_karat} size="sm" />
      </td>
      <td className="px-5 py-4 text-right font-mono text-sm text-chalk">{fmt(item.weight_grams, 2)}</td>
      <td className="px-5 py-4 text-right font-mono text-sm text-muted">{fmt(item.price_per_gram_paid, 3)}</td>
      <td className="px-5 py-4 text-right font-mono text-sm text-muted">{fmt(item.price_paid_total, 2)}</td>
      <td className="px-5 py-4 text-right font-mono text-sm text-gold-100">{fmt(item.current_value, 2)}</td>
      <td className="px-5 py-4 text-right">
        <p className={`font-mono text-sm font-semibold ${tone}`}>
          {gain >= 0 ? '+' : ''}{fmt(gain, 2)}
        </p>
        <p className={`font-mono text-xs ${tone} opacity-60`}>{fmt(item.gain_loss_pct, 2)}%</p>
      </td>
      <td className="px-5 py-4">
        <div className="flex justify-end gap-1">
          <button
            onClick={() => onEdit(item)}
            aria-label={`Edit ${item.item_name}`}
            className="rounded-chip px-2 py-1 font-display text-[10px] font-semibold uppercase tracking-stamp text-muted transition-colors hover:bg-line/40 hover:text-chalk"
          >
            Edit
          </button>
          <button
            onClick={() => onDelete(item)}
            aria-label={`Delete ${item.item_name}`}
            className="rounded-chip px-2 py-1 font-display text-[10px] font-semibold uppercase tracking-stamp text-muted transition-colors hover:bg-oxide/10 hover:text-oxide"
          >
            Delete
          </button>
        </div>
      </td>
    </tr>
  );
}
