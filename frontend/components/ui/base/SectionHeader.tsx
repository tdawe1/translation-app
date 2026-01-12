/**
 * SectionHeader - Consistent heading pattern following Data Factory design
 *
 * Pattern:
 * - Heading: text-4xl font-light tracking-tighter
 * - Label: font-mono text-xs uppercase tracking-widest (ROYGBIV accent)
 */

import { forwardRef } from "react";
import { cn } from "@/lib/utils";
import { DESIGN, type AccentColor } from "@/lib/design/tokens";

export interface SectionHeaderProps extends React.HTMLAttributes<HTMLDivElement> {
  /** Main heading title */
  title: string;
  /** Meta description label (uppercase text below title) */
  meta: string;
  /** Optional accent color for meta text (ROYGBIV) */
  accentColor?: AccentColor;
  /** Use smaller heading variant (text-2xl instead of text-4xl) */
  compact?: boolean;
  /** Hide the meta label */
  metaHidden?: boolean;
  /** Test ID for testing */
  testId?: string;
}

export const SectionHeader = forwardRef<HTMLDivElement, SectionHeaderProps>(
  (
    {
      title,
      meta,
      accentColor,
      compact = false,
      metaHidden = false,
      className = "",
      testId,
      ...props
    },
    ref
  ) => {
    const accentClass = accentColor
      ? DESIGN.colors.accent[accentColor]
      : "text-neutral-600";
    const titleClass = compact ? "text-2xl" : "text-4xl";

    return (
      <div ref={ref} className={cn("mb-6", className)} data-testid={testId} {...props}>
        <h2 className={`${titleClass} ${DESIGN.typography.weight.light} tracking-tighter mb-2 text-neutral-900`}>
          {title}
        </h2>
        {!metaHidden && (
          <p className={`${accentClass} ${DESIGN.typography.label} font-medium`}>
            {meta}
          </p>
        )}
      </div>
    );
  }
);

SectionHeader.displayName = "SectionHeader";
