const VARIANTS = {
  default: 'border-line-bright bg-ink-sunken text-muted',
  gain: 'border-patina/30 bg-patina/10 text-patina',
  loss: 'border-oxide/30 bg-oxide/10 text-oxide',
  gold: 'border-gold-600/40 bg-gold-400/10 text-gold-400',
};

export function Badge({ variant = 'default', children }) {
  return (
    <span
      className={`inline-flex items-center rounded-chip border px-2 py-0.5 font-display text-[10px] font-semibold uppercase tracking-stamp ${VARIANTS[variant]}`}
    >
      {children}
    </span>
  );
}
