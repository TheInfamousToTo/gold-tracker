/**
 * Controls are neutral. Green and red belong to the andon, so a button
 * that is merely the main action must not borrow them — weight and
 * contrast carry the hierarchy instead. `destructive` is the one
 * exception: there the red is the warning, not the emphasis.
 */
const VARIANTS = {
  primary: 'bg-chalk text-ink hover:bg-white disabled:hover:bg-chalk',
  secondary: 'border border-line-bright bg-ink-sunken text-chalk hover:border-chalk/40 hover:bg-line/40',
  ghost: 'text-muted hover:bg-line/40 hover:text-chalk',
  danger: 'text-muted hover:bg-bad/10 hover:text-bad-bright',
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
