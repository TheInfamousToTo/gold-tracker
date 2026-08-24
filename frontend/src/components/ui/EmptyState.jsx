import { Button } from './Button.jsx';

/** An empty screen is an invitation to act, so `action` is expected. */
export function EmptyState({ title, description, action }) {
  return (
    <div className="flex flex-col items-center gap-3 px-6 py-16 text-center">
      <p className="font-display text-sm font-semibold text-chalk">{title}</p>
      {description && <p className="max-w-xs text-xs leading-relaxed text-muted">{description}</p>}
      {action && (
        <div className="pt-2">
          <Button variant="secondary" size="sm" onClick={action.onClick}>
            {action.label}
          </Button>
        </div>
      )}
    </div>
  );
}
