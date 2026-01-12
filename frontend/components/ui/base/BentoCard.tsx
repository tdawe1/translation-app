/**
 * BentoCard - Strict card component enforcing Data Factory design language
 *
 * Features:
 * - Sharp corners (no border-radius)
 * - 1px border with blue-600 hover (NO shadow lift)
 * - Optional stagger animation delay
 * - Optional hover disable
 */

import { forwardRef } from "react";
import { cn } from "@/lib/utils";
import { DESIGN, type AccentColor } from "@/lib/design/tokens";

export interface BentoCardProps extends React.HTMLAttributes<HTMLDivElement> {
  /** Optional accent color for the label (ROYGBIV) */
  accentColor?: AccentColor;
  /** Disable hover effect */
  hoverDisabled?: boolean;
  /** Animation index for stagger delay (0-4) */
  staggerIndex?: number;
  /** Use smaller padding */
  compact?: boolean;
  /** Test ID for testing */
  testId?: string;
}

export const BentoCard = forwardRef<HTMLDivElement, BentoCardProps>(
  (
    {
      children,
      className = "",
      accentColor,
      hoverDisabled = false,
      staggerIndex = 0,
      compact = false,
      testId,
      ...props
    },
    ref
  ) => {
    const hoverClass = hoverDisabled ? "" : "hover:border-blue-600";
    const paddingClass = compact ? "p-4" : "p-6";
    const staggerStyle = {
      animationDelay: DESIGN.getStaggerDelay(staggerIndex),
    };

    return (
      <div
        ref={ref}
        data-testid={testId}
        className={cn(
          "bento-card",
          paddingClass,
          hoverClass,
          className
        )}
        style={staggerStyle}
        {...props}
      >
        {children}
      </div>
    );
  }
);

BentoCard.displayName = "BentoCard";
