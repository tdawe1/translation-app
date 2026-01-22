"use client";

interface LastActivityProps {
  lastActivity?: string;
}

export function LastActivity({ lastActivity }: LastActivityProps) {
  if (!lastActivity) return null;

  return (
    <div className="mt-6 text-center">
      <p className="font-mono text-xs text-neutral-400">
        Last activity: {new Date(lastActivity).toLocaleString()}
      </p>
    </div>
  );
}
