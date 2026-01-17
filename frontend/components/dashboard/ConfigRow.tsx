"use client";

interface ConfigRowProps {
  label: string;
  value: string | number | boolean;
  truncate?: boolean;
}

export function ConfigRow({ label, value, truncate = false }: ConfigRowProps) {
  return (
    <div className="flex justify-between py-2 border-b border-neutral-100 last:border-0">
      <span className="text-sm text-neutral-600">{label}</span>
      <span
        className={`font-mono text-xs text-neutral-900 ${truncate ? "truncate max-w-[200px]" : ""}`}
      >
        {typeof value === "boolean" ? (value ? "Enabled" : "Disabled") : value}
      </span>
    </div>
  );
}
