export function Toast({ message, kind = 'success', onDismiss }) {
  if (!message) return null;
  const tone =
    kind === 'error' ? 'border-bad/50 bg-bad/10 text-bad-bright' : 'border-ok/50 bg-ok/10 text-ok-bright';
  const mark = kind === 'error' ? '✕' : '✓';

  return (
    <div
      role="status"
      className={`fixed bottom-6 right-6 z-50 flex items-center gap-3 rounded-lg border bg-ink-raised px-4 py-3 text-sm shadow-2xl ${tone}`}
    >
      <span aria-hidden="true" className="font-mono">{mark}</span>
      {message}
      {onDismiss && (
        <button onClick={onDismiss} className="stamp hover:text-chalk" aria-label="Dismiss">
          ✕
        </button>
      )}
    </div>
  );
}
