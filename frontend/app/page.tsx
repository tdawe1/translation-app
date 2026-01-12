/**
 * Landing Page - Public home page
 *
 * Follows Data Factory design language with hero section, feature grid, and CTA.
 */

import Link from "next/link";
import { BentoCard } from "@/components/ui/base/BentoCard";
import { SectionHeader } from "@/components/ui/base/SectionHeader";
import { Button } from "@/components/ui/base/Button";
import { DESIGN } from "@/lib/design/tokens";

export default function HomePage() {
  return (
    <main className="min-h-screen bg-neutral-50">
      {/* Hero Section */}
      <section className="pt-44 pb-24">
        <div className="max-w-4xl mx-auto px-6">
          <h1 className="text-6xl font-light tracking-tighter mb-8 text-neutral-900">
            GengoWatcher SaaS
          </h1>
          <p className="text-xl text-neutral-700 mb-12 max-w-2xl leading-relaxed">
            Multi-tenant job monitoring with per-user watcher instances. Monitor
            Gengo jobs in real-time with RSS and WebSocket feeds.
          </p>

          {/* CTA Buttons */}
          <div className="flex gap-4">
            <Link href="/auth/register">
              <Button variant="primary">Get Started</Button>
            </Link>
            <Link href="/auth/login">
              <Button variant="secondary">Sign In</Button>
            </Link>
          </div>
        </div>
      </section>

      {/* Feature Grid - Bento layout */}
      <section className="py-24 bg-white border-t border-neutral-200">
        <div className="max-w-6xl mx-auto px-6">
          <SectionHeader
            title="Platform Features"
            meta="CAPABILITIES"
            accentColor="blue"
          />

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Status Card */}
            <BentoCard
              accentColor="red"
              staggerIndex={0}
              testId="status-card"
              className="p-8"
            >
              <h2 className="font-mono text-xs uppercase tracking-widest mb-2 font-semibold text-red-600">
                Status
              </h2>
              <p className="text-2xl text-neutral-900 font-light">
                Active Development
              </p>
              <p className="text-sm text-neutral-600 mt-2">
                Building the future of job monitoring
              </p>
            </BentoCard>

            {/* Tech Stack Card */}
            <BentoCard
              accentColor="orange"
              staggerIndex={1}
              testId="stack-card"
              className="p-8"
            >
              <h2 className="font-mono text-xs uppercase tracking-widest mb-2 font-semibold text-orange-600">
                Technology
              </h2>
              <p className="text-2xl text-neutral-900 font-light mb-2">
                Modern Stack
              </p>
              <p className="text-sm text-neutral-600">
                Go + Next.js 16 + PostgreSQL + Redis
              </p>
            </BentoCard>

            {/* Authentication Card */}
            <BentoCard
              accentColor="yellow"
              staggerIndex={2}
              testId="auth-card"
              className="p-8"
            >
              <h2 className="font-mono text-xs uppercase tracking-widest mb-2 font-semibold text-yellow-600">
                Security
              </h2>
              <p className="text-2xl text-neutral-900 font-light mb-2">
                Enterprise Auth
              </p>
              <p className="text-sm text-neutral-600">
                JWT + httpOnly Cookies + OAuth
              </p>
            </BentoCard>

            {/* Real-time Monitoring */}
            <BentoCard
              accentColor="green"
              staggerIndex={3}
              testId="monitoring-card"
              className="p-8"
            >
              <h2 className="font-mono text-xs uppercase tracking-widest mb-2 font-semibold text-green-600">
                Real-Time
              </h2>
              <p className="text-2xl text-neutral-900 font-light mb-2">
                Instant Alerts
              </p>
              <p className="text-sm text-neutral-600">
                WebSocket push notifications for new jobs
              </p>
            </BentoCard>

            {/* Multi-Tenant */}
            <BentoCard
              accentColor="cyan"
              staggerIndex={4}
              testId="tenant-card"
              className="p-8"
            >
              <h2 className="font-mono text-xs uppercase tracking-widest mb-2 font-semibold text-cyan-600">
                Multi-Tenant
              </h2>
              <p className="text-2xl text-neutral-900 font-light mb-2">
                Isolated Watchers
              </p>
              <p className="text-sm text-neutral-600">
                Per-user RSS feeds with custom filters
              </p>
            </BentoCard>

            {/* Analytics */}
            <BentoCard
              accentColor="violet"
              staggerIndex={0}
              testId="analytics-card"
              className="p-8"
            >
              <h2 className="font-mono text-xs uppercase tracking-widest mb-2 font-semibold text-violet-600">
                Analytics
              </h2>
              <p className="text-2xl text-neutral-900 font-light mb-2">
                Job Insights
              </p>
              <p className="text-sm text-neutral-600">
                Track earnings, acceptance rates, and trends
              </p>
            </BentoCard>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-24 bg-neutral-900 text-white border-t border-neutral-800">
        <div className="max-w-4xl mx-auto px-6 text-center">
          <h2 className="text-4xl font-light tracking-tighter mb-6">
            Ready to monitor smarter?
          </h2>
          <p className="text-xl text-neutral-400 mb-12 max-w-2xl mx-auto">
            Join the waitlist and be first to access GengoWatcher SaaS when we launch.
          </p>
          <div className="flex gap-4 justify-center">
            <Link href="/auth/register">
              <Button variant="primary">Create Free Account</Button>
            </Link>
            <Link href="/auth/login">
              <Button variant="secondary">Sign In</Button>
            </Link>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-12 bg-white border-t border-neutral-200">
        <div className="max-w-6xl mx-auto px-6">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <p className="text-sm text-neutral-600">
              © {new Date().getFullYear()} GengoWatcher SaaS. All rights reserved.
            </p>
            <nav className="flex gap-6">
              <Link
                href="/auth/login"
                className="text-sm text-neutral-600 hover:text-blue-600 transition-colors duration-150"
              >
                Sign In
              </Link>
              <Link
                href="/auth/register"
                className="text-sm text-neutral-600 hover:text-blue-600 transition-colors duration-150"
              >
                Register
              </Link>
            </nav>
          </div>
        </div>
      </footer>
    </main>
  );
}
