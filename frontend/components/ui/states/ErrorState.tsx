/**
 * ErrorState - Recoverable error states with retry action
 *
 * Shows when something goes wrong with helpful next steps.
 */

import { cn } from "@/lib/utils";

export interface ErrorStateProps {
  /** Error title */
  title?: string;
  /** Error message description */
  message: string;
  /** Primary retry action */
  retry?: {
    label: string;
    onClick: () => void;
  };
  /** Optional secondary action (e.g., dismiss, go back) */
  secondaryAction?: {
    label: string;
    onClick: () => void;
  };
  /** Custom className */
  className?: string;
  /** Compact variant for smaller spaces */
  compact?: boolean;
}

export const ErrorState = ({
  title = "Something went wrong",
  message,
  retry,
  secondaryAction,
  className,
  compact = false,
}: ErrorStateProps) => {
  const paddingClass = compact ? "p-4" : "p-6";

  return (
    <div
      className={cn(
        "bg-red-50 border border-red-200",
        paddingClass,
        className
      )}
    >
      <div className="flex items-start gap-3">
        {/* Error icon */}
        <span
          className="text-red-500 flex-shrink-0"
          aria-hidden="true"
        >
          <svg
            className="w-5 h-5"
            fill="currentColor"
            viewBox="0 0 20 20"
          >
            <path
              fillRule="evenodd"
              d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
              clipRule="evenodd"
            />
          </svg>
        </span>

        {/* Error content */}
        <div className="flex-1">
          {title && (
            <h3 className="text-sm font-medium text-red-800 mb-1">
              {title}
            </h3>
          )}
          <p className="text-sm text-red-600">{message}</p>

          {/* Actions */}
          {(retry || secondaryAction) && (
            <div className="flex gap-3 mt-3">
              {retry && (
                <button
                  onClick={retry.onClick}
                  className="px-4 py-2 bg-red-600 text-white text-sm hover:bg-red-700 transition-colors duration-150"
                >
                  {retry.label}
                </button>
              )}
              {secondaryAction && (
                <button
                  onClick={secondaryAction.onClick}
                  className="px-4 py-2 border border-red-300 text-red-700 text-sm hover:border-red-400 transition-colors duration-150"
                >
                  {secondaryAction.label}
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
