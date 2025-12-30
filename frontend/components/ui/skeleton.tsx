/**
 * Skeleton Component - Loading placeholder for content
 */

export function Skeleton({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={`animate-pulse rounded bg-neutral-200 ${className || ""}`}
      {...props}
    />
  );
}
