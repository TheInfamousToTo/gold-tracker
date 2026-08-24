/**
 * Status chip. The variants are the andon states, not a palette: pick
 * the one that matches the condition, never the one that looks right.
 */
const VARIANTS = {
  idle: 'andon-idle',
  ok: 'andon-ok',
  warn: 'andon-warn',
  bad: 'andon-bad',
};

export function Badge({ variant = 'idle', children }) {
  return <span className={`andon ${VARIANTS[variant] || VARIANTS.idle}`}>{children}</span>;
}
