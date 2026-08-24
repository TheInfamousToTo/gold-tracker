/**
 * The base surface. `title` renders a stamped header rule; `actions`
 * sits opposite it.
 */
export function Card({ title, actions, children, className = '', padded = true }) {
  return (
    <section className={`rounded-lg border border-line bg-ink-raised ${className}`}>
      {(title || actions) && (
        <header className="flex items-center justify-between gap-4 border-b border-line px-5 py-3">
          {title && <h2 className="stamp">{title}</h2>}
          {actions}
        </header>
      )}
      <div className={padded ? 'p-5' : ''}>{children}</div>
    </section>
  );
}
