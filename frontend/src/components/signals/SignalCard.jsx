import { fmt } from '../../lib/format.js';
import { Badge } from '../ui/Badge.jsx';

const TONE = { BUY: 'gain', SELL: 'loss', HOLD: 'default' };

export function SignalCard({ signal }) {
  return (
    <article className="rounded-lg border border-line bg-ink-raised p-5">
      <header className="mb-3 flex flex-wrap items-center gap-3">
        <Badge variant={TONE[signal.signal_type] || 'default'}>{signal.signal_type}</Badge>
        <span className="stamp">{new Date(signal.signal_date).toLocaleString()}</span>
        {signal.price_at_signal != null && (
          <span className="ml-auto font-mono text-xs text-gold-100">
            {fmt(signal.price_at_signal, 3)} BHD/g
          </span>
        )}
      </header>
      <p className="text-sm leading-relaxed text-chalk/90">{signal.reasoning}</p>
      <footer className="mt-3 flex gap-3 border-t border-line pt-3">
        <span className="stamp">{signal.source}</span>
        {signal.model && <span className="stamp">{signal.model}</span>}
      </footer>
    </article>
  );
}
