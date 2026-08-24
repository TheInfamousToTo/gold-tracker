import { SignalCard } from './SignalCard.jsx';
import { Card } from '../ui/Card.jsx';
import { Button } from '../ui/Button.jsx';
import { EmptyState } from '../ui/EmptyState.jsx';

export function SignalPanel({ signals, status, generating, error, onGenerate }) {
  const enabled = !!status?.enabled;

  return (
    <Card
      title="Analysis"
      actions={
        enabled ? (
          <Button size="sm" onClick={onGenerate} loading={generating} loadingLabel="Analysing">
            Analyse now
          </Button>
        ) : (
          <span className="stamp">Not configured</span>
        )
      }
    >
      {!enabled && (
        <p className="mb-4 text-xs leading-relaxed text-muted">
          Set <code className="font-mono text-gold-100">AI_ENABLED=true</code> on the API to generate
          buy and sell recommendations from your price history and holdings.
        </p>
      )}

      {(error || status?.last_error) && (
        <p className="mb-4 rounded-chip border border-oxide/30 bg-oxide/10 px-3 py-2 text-xs text-oxide">
          {error || status.last_error}
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
