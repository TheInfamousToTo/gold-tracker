import { fmt } from '../../lib/format.js';
import { Skeleton } from '../ui/Skeleton.jsx';

/**
 * Two of these four numbers are facts (what you put in, what it is
 * worth) and two are a verdict (are you up or down). Only the verdict
 * gets colour, and it gets it on the card edge as well as the figure,
 * so the board reads from across the room. The arrow repeats the same
 * message in shape, for anyone the colour does not reach.
 */
const TONES = {
  neutral: { text: 'text-chalk', edge: 'border-l-line-bright', mark: '' },
  ok: { text: 'text-ok-bright', edge: 'border-l-ok', mark: '▲' },
  bad: { text: 'text-bad-bright', edge: 'border-l-bad', mark: '▼' },
};

function Stat({ label, value, unit, tone = 'neutral', loading }) {
  const t = TONES[tone];
  return (
    <div className={`rounded-lg border border-line border-l-4 bg-ink-raised px-5 py-4 ${t.edge}`}>
      <p className="stamp">{label}</p>
      {loading ? (
        <Skeleton className="mt-3 h-7 w-28" />
      ) : (
        <p className="mt-2 flex items-baseline gap-1.5">
          {t.mark && (
            <span aria-hidden="true" className={`text-sm ${t.text}`}>
              {t.mark}
            </span>
          )}
          <span className={`font-mono text-2xl font-semibold ${t.text}`}>{value}</span>
          <span className="stamp">{unit}</span>
        </p>
      )}
    </div>
  );
}

export function StatGrid({ totals, loading }) {
  const gainLoss = totals.total_gain_loss || 0;
  const up = gainLoss >= 0;
  const tone = up ? 'ok' : 'bad';
  const sign = up ? '+' : '';

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <Stat label="Invested" value={fmt(totals.total_paid, 2)} unit="BHD" loading={loading} />
      <Stat label="Market value" value={fmt(totals.total_value, 2)} unit="BHD" loading={loading} />
      <Stat
        label={up ? 'Gain' : 'Loss'}
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
