import { fmt } from '../../lib/format.js';
import { VERDICT_TONE, verdictLabel, recordableBuy, confidencePct } from '../../lib/advice.js';
import { Badge } from '../ui/Badge.jsx';
import { Button } from '../ui/Button.jsx';

const EDGE = { ok: 'border-l-ok', warn: 'border-l-warn', bad: 'border-l-bad', idle: 'border-l-line-bright' };

/**
 * One verdict: an analysis of a metal, or a review of a plan the owner
 * typed. Hold — the normal state, where nothing needs doing — stays
 * colourless so the calls that do need acting on stand out.
 */
export function SignalCard({ signal, onRecordBuy }) {
  const type = signal.signal_type;
  const tone = VERDICT_TONE[type] || 'idle';
  const isReview = signal.kind === 'review';
  const action = isReview ? signal.suggested_action : type;
  const confidence = confidencePct(signal.confidence);
  const buy = recordableBuy(signal);
  const levels = signal.levels || {};

  return (
    <article className={`rounded-lg border border-line border-l-4 bg-ink-raised p-5 ${EDGE[tone]}`}>
      <header className="mb-3 flex flex-wrap items-center gap-3">
        <Badge variant={tone}>{verdictLabel(type)}</Badge>
        {isReview && <span className="stamp text-chalk">Plan review</span>}
        {/* Which market the call is about. Neutral type: naming a metal
            is not a status, and the verdict beside it carries the colour. */}
        {signal.metal && <span className="stamp text-chalk">{signal.metal}</span>}
        {confidence != null && <span className="stamp">{confidence}% confident</span>}
        {signal.deviates && <span className="stamp text-warn-bright">differs from rules</span>}
        <span className="stamp">{new Date(signal.signal_date).toLocaleString()}</span>
        {signal.price_at_signal != null && (
          <span className="ml-auto font-mono text-xs text-muted">{fmt(signal.price_at_signal, 3)} BHD/g</span>
        )}
      </header>

      {isReview && signal.plan_text && (
        <p className="mb-3 border-l-2 border-line-bright pl-3 text-xs italic text-muted">“{signal.plan_text}”</p>
      )}

      {action && action !== 'HOLD' && signal.amount_bhd > 0 && (
        <p className="mb-3 font-mono text-sm text-chalk">
          {isReview && <span className="stamp mr-2">Suggested</span>}
          {verdictLabel(action)} {fmt(signal.amount_bhd, 2)} BHD
          {signal.amount_grams > 0 && <span className="text-muted"> ≈ {fmt(signal.amount_grams, 3)} g</span>}
        </p>
      )}

      <p className="text-sm leading-relaxed text-chalk/90">{signal.reasoning}</p>

      {signal.key_factors?.length > 0 && (
        <ul className="mt-3 flex flex-wrap gap-2">
          {signal.key_factors.map((f) => (
            <li key={f} className="rounded-chip border border-line bg-ink-sunken px-2 py-1 text-[11px] text-muted">
              {f}
            </li>
          ))}
        </ul>
      )}

      {(levels.buy_more_below || levels.cut_loss_below) && (
        <p className="mt-3 flex flex-wrap gap-4 font-mono text-xs text-muted">
          {levels.buy_more_below && <span>buy more below {fmt(levels.buy_more_below, 3)}</span>}
          {levels.cut_loss_below && <span>stop-loss below {fmt(levels.cut_loss_below, 3)}</span>}
        </p>
      )}

      {signal.news?.length > 0 && (
        <ul className="mt-3 space-y-1 border-t border-line pt-3">
          {signal.news.map((n) => (
            <li key={n.url} className="flex items-baseline gap-2 text-xs">
              <span className={`andon ${n.impact === 'bullish' ? 'andon-ok' : n.impact === 'bearish' ? 'andon-bad' : 'andon-idle'}`}>
                {n.impact}
              </span>
              <a href={n.url} target="_blank" rel="noreferrer noopener" className="text-chalk underline-offset-2 hover:underline">
                {n.title}
              </a>
              <span className="text-muted">{[n.source, n.date].filter(Boolean).join(' · ')}</span>
            </li>
          ))}
        </ul>
      )}

      <footer className="mt-3 flex flex-wrap items-center gap-3 border-t border-line pt-3">
        <span className="stamp">{signal.source}</span>
        {signal.model && <span className="stamp">{signal.model}</span>}
        {signal.kind && !signal.news_checked && <span className="stamp">no news checked</span>}
        {buy && onRecordBuy && (
          <span className="ml-auto">
            <Button size="sm" variant="secondary" onClick={() => onRecordBuy(buy)}>
              I did it
            </Button>
          </span>
        )}
      </footer>
    </article>
  );
}
