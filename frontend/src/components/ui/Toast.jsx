export function Toast({ message, kind = 'success', onDismiss }) {
  if (!message) return null;
  const tone = kind === 'error' ? 'border-oxide/40 text-oxide' : 'border-patina/40 text-patina';

  return (
    <div
      role="status"
      className={`fixed bottom-6 right-6 z-50 flex items-center gap-3 rounded-lg border bg-ink-raised px-4 py-3 text-sm shadow-2xl ${tone}`}
    >
      {message}
      {onDismiss && (
        <button onClick={onDismiss} className="stamp hover:text-chalk" aria-label="Dismiss">
          ✕
        </button>
      )}
    </div>
  );
}
