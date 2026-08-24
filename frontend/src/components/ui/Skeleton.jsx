export function Skeleton({ className = 'h-4 w-24' }) {
  return <div className={`animate-pulse rounded-chip bg-line ${className}`} />;
}
