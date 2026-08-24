import { fmt } from '../../lib/format.js';
import { Badge } from '../ui/Badge.jsx';

/**
 * A recommendation is a call to act, so it is coloured like one: buy is
 * a green light, sell is a red one, and hold — the normal state, where
 * nothing needs doing — is deliberately colourless. If every card were
 * coloured, none of them would signal anything.
 */
const TONE = { BUY: 'ok', SELL: 'bad', HOLD: 'idle' };
const EDGE = { BUY: 'border-l-ok', SELL: 'border-l-bad', HOLD: 'border-l-line-bright' };

export function SignalCard({ signal }) {
  const type = signal.signal_type;

  return (
    <article
      className={`rounded-lg border border-line border-l-4 bg-ink-raised p-5 ${EDGE[type] || EDGE.HOLD}`}
    >
      <header className="mb-3 flex flex-wrap items-center gap-3">
        <Badge variant={TONE[type] || 'idle'}>{type}</Badge>
        <span className="stamp">{new Date(signal.signal_date).toLocaleString()}</span>
        {signal.price_at_signal != null && (
          <span className="ml-auto font-mono text-xs text-muted">
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
