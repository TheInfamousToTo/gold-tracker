import { SignalCard } from './SignalCard.jsx';
import { Card } from '../ui/Card.jsx';
import { Button } from '../ui/Button.jsx';
import { Badge } from '../ui/Badge.jsx';
import { EmptyState } from '../ui/EmptyState.jsx';

export function SignalPanel({ signals, status, generating, error, onGenerate }) {
  const enabled = !!status?.enabled;
  const failed = error || status?.last_error;

  return (
    <Card
      title="Analysis"
      actions={
        enabled ? (
          <Button size="sm" onClick={onGenerate} loading={generating} loadingLabel="Analysing">
            Analyse now
          </Button>
        ) : (
          /* Switched off is not broken — amber, the state that wants
             attention but is not yet a failure. */
          <Badge variant="warn">Not configured</Badge>
        )
      }
    >
      {!enabled && (
        <p className="mb-4 text-xs leading-relaxed text-muted">
          Set <code className="font-mono text-chalk">AI_ENABLED=true</code> on the API to generate
          buy and sell recommendations from your price history and holdings.
        </p>
      )}

      {failed && (
        <p className="mb-4 flex items-center gap-2 rounded-chip border border-bad/30 border-l-4 border-l-bad bg-bad/10 px-3 py-2 text-xs text-bad-bright">
          <span className="andon andon-bad">Failed</span>
          {failed}
        </p>
      )}

      {signals.length === 0 ? (
        <EmptyState
          title="No analysis yet"
          description={
            enabled
              ? 'Run an analysis to get a buy, sell, or hold call on your current position.'
              : 'Recommendations appear here once analysis is switched on.'
          }
          action={enabled ? { label: 'Analyse now', onClick: onGenerate } : undefined}
        />
      ) : (
        <div className="space-y-4">
          {signals.map((s) => <SignalCard key={s.id} signal={s} />)}
        </div>
      )}
    </Card>
  );
}
