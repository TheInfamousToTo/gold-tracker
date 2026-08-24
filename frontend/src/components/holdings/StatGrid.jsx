import { fmt } from '../../lib/format.js';
import { Skeleton } from '../ui/Skeleton.jsx';

function Stat({ label, value, unit, tone = 'default', loading }) {
  const tones = {
    default: 'text-chalk',
    gain: 'text-patina',
    loss: 'text-oxide',
    gold: 'text-gold-400',
  };
  return (
    <div className="rounded-lg border border-line bg-ink-raised px-5 py-4">
      <p className="stamp">{label}</p>
      {loading ? (
        <Skeleton className="mt-3 h-7 w-28" />
      ) : (
        <p className="mt-2 flex items-baseline gap-1.5">
          <span className={`font-mono text-2xl font-semibold ${tones[tone]}`}>{value}</span>
          <span className="stamp">{unit}</span>
        </p>
      )}
    </div>
  );
}

export function StatGrid({ totals, loading }) {
  const gainLoss = totals.total_gain_loss || 0;
  const tone = gainLoss >= 0 ? 'gain' : 'loss';
  const sign = gainLoss >= 0 ? '+' : '';

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <Stat label="Invested" value={fmt(totals.total_paid, 2)} unit="BHD" loading={loading} />
      <Stat label="Market value" value={fmt(totals.total_value, 2)} unit="BHD" tone="gold" loading={loading} />
      <Stat
        label="Gain / loss"
        value={`${sign}${fmt(gainLoss, 2)}`}
        unit="BHD"
        tone={tone}
        loading={loading}
      />
      <Stat
        label="Return"
        value={`${sign}${fmt(totals.total_gain_loss_pct, 2)}`}
        unit="%"
        tone={tone}
        loading={loading}
      />
    </div>
  );
}
