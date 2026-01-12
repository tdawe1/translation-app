/**
 * EmptyState - Meaningful zero-state designs
 *
 * Shows when there's no data to display with helpful guidance.
 */

import { cn } from "@/lib/utils";

export interface EmptyStateProps {
  /** Icon or SVG element to display */
  icon?: React.ReactNode;
  /** Main heading text */
  title: string;
  /** Supporting description */
  description?: string;
  /** Optional action button */
  action?: {
    label: string;
    onClick: () => void;
    variant?: "primary" | "secondary";
  };
  /** Custom className */
  className?: string;
  /** Compact variant for smaller spaces */
  compact?: boolean;
}

const DefaultIcon = () => (
  <svg
    className="w-12 h-12 text-neutral-300"
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={1.5}
      d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
    />
  </svg>
);

export const EmptyState = ({
  icon,
  title,
  description,
  action,
  className,
  compact = false,
}: EmptyStateProps) => {
  const paddingClass = compact ? "py-8" : "py-12";

  return (
    <div className={cn("text-center", paddingClass, className)}>
      {icon ? (
        <div className="flex justify-center mb-4">{icon}</div>
      ) : (
        <div className="flex justify-center mb-4">
          <DefaultIcon />
        </div>
      )}
      <h3 className="text-lg font-medium text-neutral-900 mb-1">
        {title}
      </h3>
      {description && (
        <p className="text-sm text-neutral-500 mb-4 max-w-sm mx-auto">
          {description}
        </p>
      )}
      {action && (
        <button
          onClick={action.onClick}
          className={cn(
            "px-6 py-2 text-sm transition-colors duration-150",
            action.variant === "secondary"
              ? "border border-neutral-300 text-neutral-900 hover:border-blue-600"
              : "bg-neutral-900 text-white hover:bg-blue-600"
          )}
        >
          {action.label}
        </button>
      )}
    </div>
  );
};
