"use client";

import { SectionHeader } from "@/components/ui/base/SectionHeader";
import type { AccentColor } from "@/lib/design/tokens";

interface DashboardContentHeaderProps {
  title?: string;
  meta?: string;
  accentColor?: AccentColor;
}

export function DashboardContentHeader({
  title = "Dashboard",
  meta = "WELCOME BACK",
  accentColor = "blue",
}: DashboardContentHeaderProps) {
  return (
    <SectionHeader
      title={title}
      meta={meta}
      accentColor={accentColor}
    />
  );
}
