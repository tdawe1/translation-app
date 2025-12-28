import Link from "next/link";

export default function HomePage() {
  return (
    <main className="min-h-screen bg-neutral-50 py-24">
      <div className="max-w-4xl mx-auto px-6">
        <h1 className="text-6xl font-light tracking-tighter mb-8">
          GengoWatcher SaaS
        </h1>
        <p className="text-xl text-neutral-600 mb-12 max-w-2xl">
          Multi-tenant job monitoring with per-user watcher instances. Monitor
          Gengo jobs in real-time with RSS and WebSocket feeds.
        </p>

        <div className="grid grid-cols-3 gap-6 mb-12">
          <div className="bento-card p-8">
            <h2 className="text-red-600 font-mono text-xs uppercase tracking-widest mb-2">
              Status
            </h2>
            <p className="text-2xl">In Development</p>
          </div>
          <div className="bento-card p-8">
            <h2 className="text-orange-600 font-mono text-xs uppercase tracking-widest mb-2">
              Stack
            </h2>
            <p className="text-sm">Go + Next.js 16 + PostgreSQL</p>
          </div>
          <div className="bento-card p-8">
            <h2 className="text-yellow-600 font-mono text-xs uppercase tracking-widest mb-2">
              Auth
            </h2>
            <p className="text-sm">JWT + httpOnly Cookies</p>
          </div>
        </div>

        {/* CTA Buttons */}
        <div className="flex gap-4">
          <Link
            href="/auth/register"
            className="px-6 py-3 bg-neutral-900 text-white text-sm transition-colors duration-150 hover:bg-blue-600"
          >
            Get Started
          </Link>
          <Link
            href="/auth/login"
            className="px-6 py-3 bg-white border border-neutral-200 text-sm transition-colors duration-150 hover:border-blue-500"
          >
            Sign In
          </Link>
        </div>
      </div>
    </main>
  );
}
