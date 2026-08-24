const VARIANTS = {
  primary: 'bg-gold-400 text-ink hover:bg-gold-100 disabled:hover:bg-gold-400',
  secondary: 'border border-line-bright bg-ink-sunken text-chalk hover:border-gold-600 hover:text-gold-100',
  ghost: 'text-muted hover:bg-line/40 hover:text-chalk',
  danger: 'text-muted hover:bg-oxide/10 hover:text-oxide',
};

const SIZES = {
  sm: 'px-2.5 py-1.5 text-[11px]',
  md: 'px-4 py-2.5 text-xs',
  lg: 'w-full px-4 py-3 text-xs',
};

export function Button({
  variant = 'primary',
  size = 'md',
  type = 'button',
  disabled = false,
  loading = false,
  loadingLabel = 'Working',
  onClick,
  children,
  ...rest
}) {
  return (
    <button
      type={type}
      disabled={disabled || loading}
      onClick={onClick}
      className={`rounded-chip font-display font-semibold uppercase tracking-stamp transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${VARIANTS[variant]} ${SIZES[size]}`}
      {...rest}
    >
      {loading ? `${loadingLabel}…` : children}
    </button>
  );
}
