/**
 * Settings Page - Data Factory Design
 *
 * Enhanced with sidebar navigation and base components.
 * User profile and account management.
 */

"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { ProtectedRoute } from "@/components/auth/protected-route";
import { ErrorBoundary } from "@/components/error-boundary";
import { authApi, type User } from "@/lib/api";
import Link from "next/link";
import { ProfileSection } from "./profile-section";
import { OAuthSection } from "./oauth-section";
import { SettingsSidebar } from "@/components/settings";
import { SectionHeader } from "@/components/ui/base/SectionHeader";
import { BentoCard } from "@/components/ui/base/BentoCard";
import { Button } from "@/components/ui/base/Button";

export default function SettingsPage() {
  const router = useRouter();
  const { user, setUser, clear } = useAuthStore();
  const [isLoading, setIsLoading] = useState(false);
  const [activeSection, setActiveSection] = useState("profile");

  useEffect(() => {
    const syncSection = () => {
      const hash = window.location.hash.replace("#", "");
      setActiveSection(hash || "profile");
    };

    syncSection();
    window.addEventListener("hashchange", syncSection);
    return () => window.removeEventListener("hashchange", syncSection);
  }, []);

  // Fetch fresh user data to get OAuth accounts
  useEffect(() => {
    const fetchUser = async () => {
      const freshUser = await authApi.me();
      if (freshUser) {
        setUser(freshUser);
      }
      // If freshUser is null, user was redirected to login or session expired
      // The ProtectedRoute component will handle the redirect
    };
    fetchUser();
  }, [setUser]);

  const handleLogout = async () => {
    try {
      await authApi.logout();
    } catch (err) {
      // Continue with logout even if API call fails
    } finally {
      sessionStorage.removeItem("access_token");
      clear();
      router.push("/");
    }
  };

  return (
    <ProtectedRoute>
      <ErrorBoundary>
        <main id="main-content" className="min-h-screen bg-neutral-50">
          {/* Header */}
          <header className="bg-white border-b border-neutral-200">
            <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <Link
                  href="/dashboard"
                  className="font-mono text-xs text-neutral-600 uppercase tracking-widest hover:text-blue-600 transition-colors duration-150 font-medium"
                >
                  ← Dashboard
                </Link>
                <h1 className="text-xl font-light tracking-tighter text-neutral-900">
                  Settings
                </h1>
              </div>
              <div className="flex items-center gap-4">
                <span className="hidden sm:block font-mono text-xs text-neutral-600 uppercase tracking-widest font-medium">
                  {user?.email}
                </span>
              </div>
            </div>
          </header>

          {/* Settings Content */}
          <div className="max-w-6xl mx-auto px-6 py-12">
            <SectionHeader
              title="Settings"
              meta="ACCOUNT MANAGEMENT"
              accentColor="blue"
            />

            <div className="flex flex-col md:flex-row gap-8 mt-8">
              {/* Sidebar Navigation */}
              <aside className="md:w-48 shrink-0">
                <SettingsSidebar activeSection={activeSection} />
              </aside>

              {/* Settings Sections */}
              <div className="flex-1 space-y-8">
                {/* Profile Section */}
                <section
                  id="profile"
                  aria-labelledby="profile-heading"
                  className="scroll-mt-24"
                >
                  <BentoCard
                    accentColor="blue"
                    staggerIndex={0}
                    testId="profile-card"
                    className="p-6"
                  >
                    <h2 id="profile-heading" className="font-mono text-xs uppercase tracking-widest text-blue-600 mb-4 font-semibold">
                      Profile
                    </h2>
                    <ProfileSection user={user} isLoading={isLoading} setIsLoading={setIsLoading} />
                  </BentoCard>
                </section>

                <section
                  id="billing"
                  aria-labelledby="billing-heading"
                  className="scroll-mt-24"
                >
                  <BentoCard
                    accentColor="orange"
                    staggerIndex={1}
                    testId="billing-card"
                    className="p-6"
                  >
                    <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
                      <div>
                        <h2 id="billing-heading" className="font-mono text-xs uppercase tracking-widest text-orange-600 mb-4 font-semibold">
                          Billing
                        </h2>
                        <h3 className="text-lg font-medium text-neutral-900">Free workspace with paid upgrade paths</h3>
                        <p className="mt-2 max-w-2xl text-sm text-neutral-600 leading-relaxed">
                          Manage launch pricing, open secure Stripe checkout, and keep billing attached to <span data-testid="settings-billing-email" className="font-mono text-neutral-900">{user?.email}</span>.
                        </p>
                      </div>
                      <div className="flex flex-wrap gap-3">
                        <Link href="/pricing" data-testid="settings-billing-open-pricing-link">
                          <Button testId="settings-billing-open-pricing-button" variant="primary">
                            Open pricing
                          </Button>
                        </Link>
                      </div>
                    </div>
                  </BentoCard>
                </section>

                {/* Connected Accounts Section */}
                <section
                  id="accounts"
                  aria-labelledby="accounts-heading"
                  className="scroll-mt-24"
                >
                  <BentoCard
                    accentColor="green"
                    staggerIndex={1}
                    testId="accounts-card"
                    className="p-6"
                  >
                    <h2 id="accounts-heading" className="font-mono text-xs uppercase tracking-widest text-green-600 mb-4 font-semibold">
                      Connected Accounts
                    </h2>
                    <OAuthSection user={user} />
                  </BentoCard>
                </section>

                {/* Danger Zone */}
                <section
                  id="danger"
                  aria-labelledby="danger-heading"
                  className="scroll-mt-24"
                >
                  <BentoCard
                    accentColor="red"
                    staggerIndex={2}
                    testId="danger-card"
                    className="p-6"
                  >
                    <h2 id="danger-heading" className="font-mono text-xs uppercase tracking-widest text-red-600 mb-4 font-semibold">
                      Danger Zone
                    </h2>

                    <div className="flex items-center justify-between">
                      <div>
                        <h3 className="text-lg font-medium mb-1 text-neutral-900">Sign Out</h3>
                        <p className="text-sm text-neutral-600">Sign out of your account on this device</p>
                      </div>
                      <Button
                        onClick={handleLogout}
                        variant="danger"
                      >
                        Sign Out
                      </Button>
                    </div>
                  </BentoCard>
                </section>
              </div>
            </div>
          </div>
        </main>
      </ErrorBoundary>
    </ProtectedRoute>
  );
}
