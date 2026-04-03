import Link from "next/link";
import { BentoCard } from "@/components/ui/base/BentoCard";
import { SectionHeader } from "@/components/ui/base/SectionHeader";
import { Button } from "@/components/ui/base/Button";

export default function HomePage() {
  const featureCards = [
    {
      id: "watcher",
      label: "Monitoring",
      title: "Watcher control built for speed",
      body: "Start, stop, and reconfigure job monitoring without digging through multiple tools.",
    },
    {
      id: "realtime",
      label: "Realtime",
      title: "A cleaner signal feed",
      body: "Track what happened last, what is running now, and what needs review next.",
    },
    {
      id: "review",
      label: "Review",
      title: "Translation jobs in one review loop",
      body: "Move from job detection to translation review without losing the operational context.",
    },
  ];

  return (
    <main className="min-h-screen bg-neutral-50">
      <header className="sticky top-0 z-20 border-b border-neutral-200 bg-white/95 backdrop-blur-sm">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
          <Link href="/" data-testid="home-brand-link" className="text-xl font-light tracking-tighter text-neutral-900">
            GengoWatcher
          </Link>
          <nav className="flex items-center gap-6">
            <Link href="/pricing" data-testid="home-nav-pricing-link" className="font-mono text-xs uppercase tracking-widest text-neutral-500 hover:text-neutral-900">
              Pricing
            </Link>
            <Link href="/auth/login" data-testid="home-nav-login-link" className="font-mono text-xs uppercase tracking-widest text-neutral-500 hover:text-neutral-900">
              Sign In
            </Link>
            <Link href="/auth/register" data-testid="home-nav-register-link">
              <Button testId="home-nav-get-started-button" variant="primary" size="sm" className="rounded-none">
                Get Started
              </Button>
            </Link>
          </nav>
        </div>
      </header>

      <section className="border-b border-neutral-200 bg-[radial-gradient(circle_at_top_left,_rgba(37,99,235,0.12),_transparent_30%),linear-gradient(180deg,_white_0%,_#f8fafc_100%)] px-6 py-20 lg:py-28">
        <div className="mx-auto grid max-w-6xl gap-10 lg:grid-cols-12 lg:items-center">
          <div className="lg:col-span-7">
            <p data-testid="home-hero-overline" className="mb-4 font-mono text-xs uppercase tracking-[0.3em] text-blue-700">
              Launch-ready monitoring SaaS
            </p>
            <h1 data-testid="home-hero-title" className="max-w-4xl text-4xl font-semibold tracking-tight text-neutral-900 sm:text-5xl lg:text-6xl">
              Finish the monitoring workflow before it costs your team speed.
            </h1>
            <p data-testid="home-hero-subtitle" className="mt-6 max-w-2xl text-base leading-relaxed text-neutral-600 md:text-lg">
              GengoWatcher brings account access, watcher control, translation review, and paid plans into one sharp, dependable launch surface.
            </p>

            <div className="mt-10 flex flex-wrap gap-4">
              <Link href="/auth/register" data-testid="home-hero-register-link">
                <Button testId="home-hero-register-button" variant="primary" className="rounded-none">
                  Create account
                </Button>
              </Link>
              <Link href="/pricing" data-testid="home-hero-pricing-link">
                <Button testId="home-hero-pricing-button" variant="secondary" className="rounded-none">
                  View pricing
                </Button>
              </Link>
            </div>
          </div>

          <div className="grid gap-4 lg:col-span-5">
            <div data-testid="home-hero-visual-card" className="overflow-hidden border border-neutral-900 bg-neutral-900 p-0 text-white">
              <div className="grid gap-0 md:grid-cols-[1.1fr_0.9fr]">
                <div className="p-8">
                  <p className="font-mono text-xs uppercase tracking-[0.25em] text-neutral-400">Operator view</p>
                  <div className="mt-8 space-y-4">
                    <div className="border border-white/15 p-4">
                      <p className="font-mono text-xs uppercase tracking-widest text-neutral-400">Watcher status</p>
                      <p className="mt-2 text-2xl font-semibold">Running</p>
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div className="border border-white/15 p-4">
                        <p className="font-mono text-xs uppercase tracking-widest text-neutral-400">Jobs found</p>
                        <p className="mt-2 text-2xl font-semibold">284</p>
                      </div>
                      <div className="border border-white/15 p-4">
                        <p className="font-mono text-xs uppercase tracking-widest text-neutral-400">Accepted</p>
                        <p className="mt-2 text-2xl font-semibold">119</p>
                      </div>
                    </div>
                  </div>
                </div>
                <div className="min-h-[280px] bg-neutral-200">
                  <img
                    data-testid="home-hero-image"
                    src="https://images.unsplash.com/photo-1762279388988-3f8abcc7dca2?crop=entropy&cs=srgb&fm=jpg&ixid=M3w3NTY2NzZ8MHwxfHNlYXJjaHwzfHxhYnN0cmFjdCUyMGdlb21ldHJpYyUyMGRhdGF8ZW58MHx8fHwxNzc1MTMyMjkxfDA&ixlib=rb-4.1.0&q=85"
                    alt="Abstract data flow visual"
                    className="h-full w-full object-cover"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="py-24 bg-white border-t border-neutral-200">
        <div className="max-w-6xl mx-auto px-6">
          <SectionHeader
            title="Platform features"
            meta="CAPABILITIES"
            accentColor="blue"
            testId="home-features-header"
          />

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {featureCards.map((card, index) => (
              <BentoCard key={card.id} accentColor="blue" staggerIndex={index} testId={`home-feature-card-${card.id}`} className="p-8">
                <h2 className="font-mono text-xs uppercase tracking-widest mb-2 font-semibold text-blue-700">
                  {card.label}
                </h2>
                <p className="text-2xl text-neutral-900 font-light">{card.title}</p>
                <p className="text-sm text-neutral-600 mt-3 leading-relaxed">{card.body}</p>
              </BentoCard>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-neutral-200 bg-neutral-900 px-6 py-20 text-white">
        <div className="mx-auto grid max-w-6xl gap-8 lg:grid-cols-12 lg:items-end">
          <div className="lg:col-span-7">
            <SectionHeader
              title="Pricing that matches launch velocity"
              meta="BILLING"
              accentColor="blue"
              testId="home-pricing-header"
              className="mb-0"
            />
            <p data-testid="home-pricing-copy" className="mt-4 max-w-2xl text-base leading-relaxed text-neutral-300">
              Keep the product simple during rollout, then upgrade through Stripe when you want a stronger operating layer.
            </p>
          </div>
          <div className="grid gap-4 lg:col-span-5 sm:grid-cols-2">
            <div data-testid="home-pricing-pro-card" className="border border-white/15 bg-white/5 p-6 text-white transition-colors duration-150 hover:border-white/40">
              <p className="font-mono text-xs uppercase tracking-widest text-blue-300">Pro</p>
              <p className="mt-3 text-3xl font-semibold">$29</p>
              <p className="mt-3 text-sm text-neutral-300">Realtime watcher control and translation review structure.</p>
            </div>
            <div data-testid="home-pricing-team-card" className="border border-white/15 bg-white/5 p-6 text-white transition-colors duration-150 hover:border-white/40">
              <p className="font-mono text-xs uppercase tracking-widest text-blue-300">Team</p>
              <p className="mt-3 text-3xl font-semibold">$79</p>
              <p className="mt-3 text-sm text-neutral-300">Broader watcher coverage with a more collaborative review setup.</p>
            </div>
          </div>
          <div className="lg:col-span-12 flex flex-wrap gap-4 pt-4">
            <Link href="/pricing" data-testid="home-pricing-page-link">
              <Button testId="home-pricing-page-button" variant="primary" className="rounded-none">
                Open pricing
              </Button>
            </Link>
            <Link href="/auth/register" data-testid="home-cta-register-link">
              <Button testId="home-cta-register-button" variant="secondary" className="rounded-none border-white/30 bg-transparent text-white hover:bg-white hover:text-neutral-900">
                Create free account
              </Button>
            </Link>
          </div>
        </div>
      </section>

      <footer className="py-12 bg-white border-t border-neutral-200">
        <div className="max-w-6xl mx-auto px-6">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <p className="text-sm text-neutral-600">
              © {new Date().getFullYear()} GengoWatcher SaaS. All rights reserved.
            </p>
            <nav className="flex gap-6">
              <Link
                href="/pricing"
                data-testid="footer-pricing-link"
                className="text-sm text-neutral-600 hover:text-blue-600 transition-colors duration-150"
              >
                Pricing
              </Link>
              <Link
                href="/auth/login"
                data-testid="footer-signin-link"
                className="text-sm text-neutral-600 hover:text-blue-600 transition-colors duration-150"
              >
                Sign In
              </Link>
              <Link
                href="/auth/register"
                data-testid="footer-register-link"
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
