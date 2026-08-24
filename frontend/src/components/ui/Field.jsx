const INPUT_CLASS =
  'w-full rounded-chip border border-line bg-ink-sunken px-3 py-2.5 text-sm text-chalk placeholder:text-muted/50 transition-colors focus:border-chalk/40 focus:outline-none';

/** Shared input styling, exported so selects and textareas match. */
export const inputClass = INPUT_CLASS;

/**
 * A labelled form row. `hint` explains, `error` corrects — and an error
 * is an abnormal condition, so it is stated in words and marked red.
 */
export function Field({ label, htmlFor, hint, error, children }) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={htmlFor} className="stamp block">
        {label}
      </label>
      {children}
      {error ? (
        <p className="text-xs text-bad-bright">{error}</p>
      ) : (
        hint && <p className="text-xs text-muted">{hint}</p>
      )}
    </div>
  );
}
