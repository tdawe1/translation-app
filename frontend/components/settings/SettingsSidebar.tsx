/**
 * SettingsSidebar - Side navigation for settings sections
 *
 * Highlights active section and allows quick navigation between settings areas.
 * Follows Data Factory design language.
 */

import Link from "next/link";
import { usePathname } from "next/navigation";

export interface SettingsSection {
  id: string;
  label: string;
  href: string;
  accentColor: "red" | "orange" | "yellow" | "green" | "cyan" | "blue" | "indigo" | "violet";
}

const SETTINGS_SECTIONS: SettingsSection[] = [
  { id: "profile", label: "Profile", href: "/settings", accentColor: "blue" },
  { id: "accounts", label: "Connected Accounts", href: "/settings#accounts", accentColor: "green" },
  { id: "danger", label: "Danger Zone", href: "/settings#danger", accentColor: "red" },
];

const ACCENT_COLORS: Record<SettingsSection["accentColor"], string> = {
  red: "text-red-600",
  orange: "text-orange-600",
  yellow: "text-yellow-600",
  green: "text-green-600",
  cyan: "text-cyan-600",
  blue: "text-blue-600",
  indigo: "text-indigo-600",
  violet: "text-violet-600",
};

interface SettingsSidebarProps {
  activeSection?: string;
}

export function SettingsSidebar({ activeSection = "profile" }: SettingsSidebarProps) {
  const pathname = usePathname();

  return (
    <nav aria-label="Settings navigation" className="shrink-0">
      <h2 className="font-mono text-xs uppercase tracking-widest text-neutral-600 mb-4 font-medium">
        Sections
      </h2>
      <ul className="space-y-1">
        {SETTINGS_SECTIONS.map((section) => {
          const isActive = section.id === activeSection;
          const accentColor = ACCENT_COLORS[section.accentColor];

          return (
            <li key={section.id}>
              <Link
                href={section.href}
                className={`block px-4 py-2 border-l-2 transition-colors duration-150 ${
                  isActive
                    ? `border-${section.accentColor}-600 bg-neutral-100 ${accentColor}`
                    : "border-transparent text-neutral-700 hover:border-neutral-300 hover:bg-neutral-50"
                }`}
              >
                <span className="font-mono text-xs uppercase tracking-widest">
                  {section.label}
                </span>
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
