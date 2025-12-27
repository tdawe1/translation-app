export default function HomePage() {
  return (
    <main className="min-h-screen bg-neutral-50 py-24">
      <div className="max-w-4xl mx-auto px-6">
        <h1 className="text-6xl font-light tracking-tighter mb-8">
          GengoWatcher SaaS
        </h1>
        <div className="grid grid-cols-3 gap-6">
          <div className="bento-card p-8">
            <h2 className="text-red-600 font-mono text-xs uppercase tracking-widest mb-2">
              Status
            </h2>
            <p className="text-2xl">Coming Soon</p>
          </div>
          <div className="bento-card p-8">
            <h2 className="text-orange-600 font-mono text-xs uppercase tracking-widest mb-2">
              Stack
            </h2>
            <p className="text-sm">Go + Next.js 16 + BetterAuth</p>
          </div>
          <div className="bento-card p-8">
            <h2 className="text-yellow-600 font-mono text-xs uppercase tracking-widest mb-2">
              Auth
            </h2>
            <p className="text-sm">BetterAuth Integration</p>
          </div>
        </div>
      </div>
    </main>
  );
}
