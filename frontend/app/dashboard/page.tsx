/**
 * Dashboard Page - Protected route showing user info
 */

"use client";

import { useAuthStore } from "@/store/auth";
import { ProtectedRoute } from "@/components/auth/protected-route";
import { authApi } from "@/lib/api";

export default function DashboardPage() {
  const user = useAuthStore((state) => state.user);
  const logout = async () => {
    await authApi.logout();
    sessionStorage.removeItem("access_token");
    window.location.href = "/";
  };

  return (
    <ProtectedRoute>
      <main className="min-h-screen bg-neutral-50">
        {/* Header */}
        <header className="bg-white border-b border-neutral-200">
          <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
            <h1 className="text-xl font-light tracking-tighter">
              GengoWatcher
            </h1>
            <div className="flex items-center gap-4">
              <span className="font-mono text-xs text-neutral-500 uppercase tracking-widest">
                {user?.email}
              </span>
              <button
                onClick={logout}
                className="font-mono text-xs text-neutral-900 uppercase tracking-widest hover:text-blue-600"
              >
                Sign Out
              </button>
            </div>
          </div>
        </header>

        {/* Dashboard Content */}
        <div className="max-w-6xl mx-auto px-6 py-12">
          <div className="mb-8">
            <h2 className="text-4xl font-light tracking-tighter mb-2">
              Dashboard
            </h2>
            <p className="text-neutral-500 font-mono text-xs uppercase tracking-widest">
              Welcome back
            </p>
          </div>

          {/* Bento Grid */}
          <div className="grid grid-cols-3 gap-6">
            <div className="bento-card p-6">
              <h3 className="text-red-600 font-mono text-xs uppercase tracking-widest mb-2">
                Watcher Status
              </h3>
              <p className="text-3xl font-light">Stopped</p>
            </div>

            <div className="bento-card p-6">
              <h3 className="text-orange-600 font-mono text-xs uppercase tracking-widest mb-2">
                Jobs Found
              </h3>
              <p className="text-3xl font-light">0</p>
            </div>

            <div className="bento-card p-6">
              <h3 className="text-yellow-600 font-mono text-xs uppercase tracking-widest mb-2">
                Earnings
              </h3>
              <p className="text-3xl font-light">$0.00</p>
            </div>

            <div className="bento-card p-6 col-span-2">
              <h3 className="text-green-600 font-mono text-xs uppercase tracking-widest mb-4">
                Watcher Configuration
              </h3>
              <div className="space-y-3">
                <div className="flex justify-between py-2 border-b border-neutral-100">
                  <span className="text-sm text-neutral-600">RSS Feed</span>
                  <span className="font-mono text-xs">Gengo Jobs RSS</span>
                </div>
                <div className="flex justify-between py-2 border-b border-neutral-100">
                  <span className="text-sm text-neutral-600">Min Reward</span>
                  <span className="font-mono text-xs">$0.00</span>
                </div>
                <div className="flex justify-between py-2 border-b border-neutral-100">
                  <span className="text-sm text-neutral-600">Max Reward</span>
                  <span className="font-mono text-xs">$999.00</span>
                </div>
              </div>
            </div>

            <div className="bento-card p-6">
              <h3 className="text-cyan-600 font-mono text-xs uppercase tracking-widest mb-4">
                Actions
              </h3>
              <button
                disabled
                className="w-full py-3 bg-neutral-900 text-white text-sm transition-colors duration-150 hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Start Watcher
              </button>
            </div>
          </div>
        </div>
      </main>
    </ProtectedRoute>
  );
}
