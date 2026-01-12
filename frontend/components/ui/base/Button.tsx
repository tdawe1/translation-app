/**
 * Button - Consistent button component following Data Factory design
 *
 * Variants:
 * - primary: bg-neutral-900 text-white hover:bg-blue-600
 * - secondary: border border-neutral-300 hover:border-blue-600
 * - danger: text-red-600 hover:border-red-600 hover:bg-red-50
 */

import { forwardRef } from "react";
import { cn } from "@/lib/utils";

export type ButtonVariant = "primary" | "secondary" | "danger";
export type ButtonSize = "sm" | "md" | "lg";

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /** Button style variant */
  variant?: ButtonVariant;
  /** Button size */
  size?: ButtonSize;
  /** Show loading state with spinner */
  loading?: boolean;
  /** Text to show when loading (defaults to "Loading...") */
  loadingText?: string;
  /** Full width button */
  fullWidth?: boolean;
  /** Test ID for testing */
  testId?: string;
}

const SIZE_CLASSES: Record<ButtonSize, string> = {
  sm: "px-4 py-2 text-xs",
  md: "px-6 py-3 text-sm",
  lg: "px-8 py-4 text-base",
};

const VARIANT_CLASSES: Record<ButtonVariant, string> = {
  primary: "bg-neutral-900 text-white hover:bg-blue-600",
  secondary: "bg-white border border-neutral-300 text-neutral-900 hover:border-blue-600",
  danger: "bg-white border border-neutral-300 text-red-600 hover:border-red-600 hover:bg-red-50",
};

// Spinner component for loading state
const Spinner = () => (
  <svg
    className="animate-spin h-4 w-4"
    xmlns="http://www.w3.org/2000/svg"
    fill="none"
    viewBox="0 0 24 24"
  >
    <circle
      className="opacity-25"
      cx="12"
      cy="12"
      r="10"
      stroke="currentColor"
      strokeWidth="4"
    />
    <path
      className="opacity-75"
      fill="currentColor"
      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
    />
  </svg>
);

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      variant = "primary",
      size = "md",
      loading = false,
      loadingText,
      fullWidth = false,
      disabled,
      className = "",
      children,
      testId,
      ...props
    },
    ref
  ) => {
    return (
      <button
        ref={ref}
        disabled={disabled || loading}
        data-testid={testId}
        className={cn(
          // Base styles
          "inline-flex items-center justify-center gap-2",
          "transition-colors duration-150",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          // Focus styles
          "focus:outline-none focus:ring-2 focus:ring-blue-600 focus:ring-offset-2",
          // Size
          SIZE_CLASSES[size],
          // Variant
          VARIANT_CLASSES[variant],
          // Full width
          fullWidth && "w-full",
          // Custom classes
          className
        )}
        {...props}
      >
        {loading ? (
          <>
            <Spinner />
            <span>{loadingText || "Loading..."}</span>
          </>
        ) : (
          children
        )}
      </button>
    );
  }
);

Button.displayName = "Button";
