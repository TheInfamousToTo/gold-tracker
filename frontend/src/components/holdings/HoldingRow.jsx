import { fmt, fmtDate } from '../../lib/format.js';
import { Hallmark } from '../ui/Hallmark.jsx';

/**
 * One line per holding. Every figure on the row is neutral except the
 * one that says whether this piece is ahead or behind, and that one is
 * repeated on the row edge — an underwater line is findable by running
 * an eye down the left margin, without reading a single number.
 */
export function HoldingRow({ item, onEdit, onDelete }) {
  const gain = item.gain_loss ?? 0;
  const up = gain >= 0;
  const tone = up ? 'text-ok-bright' : 'text-bad-bright';
  const edge = up ? 'border-l-ok' : 'border-l-bad';

  return (
    <tr className="border-t border-line transition-colors hover:bg-line/20">
      <td className={`border-l-4 px-5 py-4 ${edge}`}>
        <p className="text-sm font-medium text-chalk">{item.item_name}</p>
        <p className="mt-0.5 text-xs text-muted">
          {item.vendor || 'Source not recorded'} · {fmtDate(item.purchase_date)}
        </p>
      </td>
      <td className="px-5 py-4">
        <Hallmark
          metal={item.metal_type || 'gold'}
          karat={item.purity_karat}
          fineness={item.purity_fineness}
          size="sm"
        />
      </td>
      <td className="px-5 py-4 text-right font-mono text-sm text-chalk">{fmt(item.weight_grams, 2)}</td>
      <td className="px-5 py-4 text-right font-mono text-sm text-muted">{fmt(item.price_per_gram_paid, 3)}</td>
      <td className="px-5 py-4 text-right font-mono text-sm text-muted">{fmt(item.price_paid_total, 2)}</td>
      <td className="px-5 py-4 text-right font-mono text-sm text-chalk">{fmt(item.current_value, 2)}</td>
      <td className="px-5 py-4 text-right">
        <p className={`font-mono text-sm font-semibold ${tone}`}>
          <span aria-hidden="true" className="mr-1">{up ? '▲' : '▼'}</span>
          {up ? '+' : ''}{fmt(gain, 2)}
        </p>
        <p className={`font-mono text-xs ${tone} opacity-70`}>{fmt(item.gain_loss_pct, 2)}%</p>
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
            className="rounded-chip px-2 py-1 font-display text-[10px] font-semibold uppercase tracking-stamp text-muted transition-colors hover:bg-bad/10 hover:text-bad-bright"
          >
            Delete
          </button>
        </div>
      </td>
    </tr>
  );
}

/**
 * The same holding as a card, for phone widths where eight columns
 * cannot fit. The verdict keeps its place on the left edge.
 */
export function HoldingCard({ item, onEdit, onDelete }) {
  const gain = item.gain_loss ?? 0;
  const up = gain >= 0;
  const tone = up ? 'text-ok-bright' : 'text-bad-bright';
  const edge = up ? 'border-l-ok' : 'border-l-bad';

  return (
    <li className={`border-l-4 border-t border-t-line px-4 py-3.5 first:border-t-0 ${edge}`}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium text-chalk">{item.item_name}</p>
          <p className="mt-0.5 text-xs text-muted">{fmtDate(item.purchase_date)}</p>
        </div>
        <div className="shrink-0 text-right">
          <p className={`font-mono text-sm font-semibold ${tone}`}>
            <span aria-hidden="true" className="mr-1">{up ? '▲' : '▼'}</span>
            {up ? '+' : ''}{fmt(gain, 2)}
          </p>
          <p className={`font-mono text-xs ${tone} opacity-70`}>{fmt(item.gain_loss_pct, 2)}%</p>
        </div>
      </div>

      <dl className="mt-3 grid grid-cols-3 gap-2">
        <div>
          <dt className="stamp">Mass</dt>
          <dd className="font-mono text-sm text-chalk">{fmt(item.weight_grams, 2)} g</dd>
        </div>
        <div>
          <dt className="stamp">Buy /g</dt>
          <dd className="font-mono text-sm text-muted">{fmt(item.price_per_gram_paid, 3)}</dd>
        </div>
        <div>
          <dt className="stamp">Now /g</dt>
          <dd className="font-mono text-sm text-chalk">{item.current_price_per_gram != null ? fmt(item.current_price_per_gram, 3) : '—'}</dd>
        </div>
        <div>
          <dt className="stamp">Paid</dt>
          <dd className="font-mono text-sm text-muted">{fmt(item.price_paid_total, 2)}</dd>
        </div>
        <div>
          <dt className="stamp">Value</dt>
          <dd className="font-mono text-sm text-chalk">{fmt(item.current_value, 2)}</dd>
        </div>
      </dl>

      <div className="mt-2 flex items-center justify-between gap-1">
        <Hallmark
          metal={item.metal_type || 'gold'}
          karat={item.purity_karat}
          fineness={item.purity_fineness}
          size="sm"
        />
        <span className="flex-1" />
        <button
          onClick={() => onEdit(item)}
          aria-label={`Edit ${item.item_name}`}
          className="min-h-[40px] rounded-chip px-3 font-display text-[11px] font-semibold uppercase tracking-stamp text-muted hover:bg-line/40 hover:text-chalk"
        >
          Edit
        </button>
        <button
          onClick={() => onDelete(item)}
          aria-label={`Delete ${item.item_name}`}
          className="min-h-[40px] rounded-chip px-3 font-display text-[11px] font-semibold uppercase tracking-stamp text-muted hover:bg-bad/10 hover:text-bad-bright"
        >
          Delete
        </button>
      </div>
    </li>
  );
}
