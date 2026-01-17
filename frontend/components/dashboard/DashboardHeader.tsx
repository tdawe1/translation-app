"use client";

import Link from "next/link";
import type { User } from "@/lib/api";

interface DashboardHeaderProps {
  user: User | null;
  onLogout: () => void;
}

export function DashboardHeader({ user, onLogout }: DashboardHeaderProps) {
  return (
    <header className="bg-white border-b border-neutral-200">
      <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
        <Link href="/" className="text-xl font-light tracking-tighter text-neutral-900 hover:text-blue-600 transition-colors duration-150">
          GengoWatcher
        </Link>
        <div className="flex items-center gap-6">
          <span
            data-testid="user-email"
            className="hidden sm:block font-mono text-xs text-neutral-500 uppercase tracking-widest"
          >
            {user?.email}
          </span>
          <Link
            data-testid="settings-link"
            href="/settings"
            className="font-mono text-xs text-neutral-500 uppercase tracking-widest hover:text-blue-600 transition-colors duration-150"
          >
            Settings
          </Link>
          <button
            data-testid="sign-out-button"
            onClick={onLogout}
            className="font-mono text-xs text-neutral-900 uppercase tracking-widest hover:text-blue-600 transition-colors duration-150"
          >
            Sign Out
          </button>
        </div>
      </div>
    </header>
  );
}
