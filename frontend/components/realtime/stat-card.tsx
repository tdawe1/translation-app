/**
 * StatCard - Reusable bento card for displaying realtime statistics
 *
 * Follows the bento-card design language with ROYGBIV color labels.
 */

import React from "react";

type StatColor = "red" | "orange" | "yellow" | "green" | "cyan" | "blue" | "indigo" | "violet";

interface StatCardProps {
  label: string;
  value: string | number;
  color: StatColor;
  trend?: {
    value: number;
    direction: "up" | "down" | "neutral";
  };
  testId?: string;
}

const COLOR_CLASSES: Record<StatColor, string> = {
  red: "text-red-600",
  orange: "text-orange-600",
  yellow: "text-yellow-600",
  green: "text-green-600",
  cyan: "text-cyan-600",
  blue: "text-blue-600",
  indigo: "text-indigo-600",
  violet: "text-violet-600",
};

export const StatCard = React.memo<StatCardProps>(({ label, value, color, trend, testId }) => {
  const colorClass = COLOR_CLASSES[color];

  return (
    <div className="bento-card p-6" data-testid={testId}>
      <h3 className={`${colorClass} font-mono text-xs uppercase tracking-widest mb-2`}>
        {label}
      </h3>
      <p className="text-3xl font-light" role="status" aria-live="polite">
        {value}
      </p>
      {trend && (
        <div className="mt-2 text-xs font-mono flex items-center gap-1">
          {trend.direction === "up" && <span aria-hidden="true">↑</span>}
          {trend.direction === "down" && <span aria-hidden="true">↓</span>}
          <span className={trend.direction === "up" ? "text-green-600" : trend.direction === "down" ? "text-red-600" : "text-neutral-500"}>
            {Math.abs(trend.value)}
          </span>
        </div>
      )}
    </div>
  );
});

StatCard.displayName = "StatCard";
