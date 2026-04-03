"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { billingApi, type BillingPlan } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { Button } from "@/components/ui/base/Button";
import { BentoCard } from "@/components/ui/base/BentoCard";
import { SectionHeader } from "@/components/ui/base/SectionHeader";

const FAQS = [
  {
    question: "Can I start free and upgrade later?",
    answer:
      "Yes. You can create your account first, explore the workflow, and upgrade once you are ready to pay through Stripe.",
  },
  {
    question: "What happens after checkout?",
    answer:
      "Stripe redirects you back to this page, where the app checks payment status and confirms the session in place.",
  },
  {
    question: "Do I need an account before paying?",
    answer:
      "For this launch-ready flow, yes. Sign in or create an account first so billing can be attached to a user email.",
  },
];

function PricingContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const user = useAuthStore((state) => state.user);
  const [plans, setPlans] = useState<BillingPlan[]>([]);
  const [plansLoading, setPlansLoading] = useState(true);
  const [checkoutPlanId, setCheckoutPlanId] = useState<string | null>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [statusTone, setStatusTone] = useState<"neutral" | "success" | "warning">("neutral");

  useEffect(() => {
    const loadPlans = async () => {
      try {
        const nextPlans = await billingApi.getPlans();
        setPlans(nextPlans);
      } finally {
        setPlansLoading(false);
      }
    };

    loadPlans();
  }, []);

  useEffect(() => {
    const sessionId = searchParams.get("session_id");
    if (!sessionId) return;

    let cancelled = false;
    let attempts = 0;

    const poll = async () => {
      try {
        const response = await billingApi.getStatus(sessionId);
        if (cancelled) return;

        if (response.payment_status === "paid") {
          setStatusTone("success");
          setStatusMessage("Payment confirmed. Your plan is ready to be connected to this account.");
          return;
        }

        if (response.status === "expired") {
          setStatusTone("warning");
          setStatusMessage("This checkout session expired. Please start a new payment.");
          return;
        }

        attempts += 1;
        setStatusTone("neutral");
        setStatusMessage("Checking your payment status...");
        if (attempts < 5) {
          window.setTimeout(poll, 2000);
        }
      } catch {
        if (!cancelled) {
          setStatusTone("warning");
          setStatusMessage("We could not confirm payment yet. Refresh to check again.");
        }
      }
    };

    poll();

    return () => {
      cancelled = true;
    };
  }, [searchParams]);

  const handleCheckout = async (planId: string) => {
    if (!user?.email) {
      router.push("/auth/register");
      return;
    }

    setCheckoutPlanId(planId);
    try {
      const session = await billingApi.createCheckout(planId, user.email);
      window.location.href = session.url;
    } finally {
      setCheckoutPlanId(null);
    }
  };

  const statusClass = useMemo(() => {
    if (statusTone === "success") return "border-green-200 bg-green-50 text-green-800";
    if (statusTone === "warning") return "border-amber-200 bg-amber-50 text-amber-800";
    return "border-neutral-200 bg-white text-neutral-700";
  }, [statusTone]);

  return (
    <main id="main-content" className="min-h-screen bg-neutral-50">
      <header className="sticky top-0 z-20 border-b border-neutral-200 bg-white/95 backdrop-blur-sm">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
          <Link href="/" data-testid="pricing-home-link" className="text-xl font-light tracking-tighter text-neutral-900">
            GengoWatcher
          </Link>
          <nav className="flex items-center gap-6">
            <Link href="/" data-testid="pricing-nav-overview-link" className="font-mono text-xs uppercase tracking-widest text-neutral-500 hover:text-neutral-900">
              Overview
            </Link>
            <Link href="/dashboard" data-testid="pricing-nav-dashboard-link" className="font-mono text-xs uppercase tracking-widest text-neutral-500 hover:text-neutral-900">
              Dashboard
            </Link>
            <Link href="/settings" data-testid="pricing-nav-settings-link" className="font-mono text-xs uppercase tracking-widest text-neutral-500 hover:text-neutral-900">
              Settings
            </Link>
          </nav>
        </div>
      </header>

      <section className="border-b border-neutral-200 bg-[radial-gradient(circle_at_top_left,_rgba(17,24,39,0.12),_transparent_35%),linear-gradient(180deg,_white_0%,_#f8fafc_100%)] px-6 py-20">
        <div className="mx-auto grid max-w-6xl gap-12 md:grid-cols-12">
          <div className="md:col-span-7">
            <p data-testid="pricing-overline" className="mb-4 font-mono text-xs uppercase tracking-[0.3em] text-blue-700">
              Simple, predictable pricing
            </p>
            <h1 data-testid="pricing-title" className="max-w-3xl text-4xl font-semibold tracking-tight text-neutral-900 sm:text-5xl lg:text-6xl">
              Launch plans built around faster monitoring and smoother translation review.
            </h1>
            <p data-testid="pricing-subtitle" className="mt-6 max-w-2xl text-base leading-relaxed text-neutral-600 md:text-lg">
              Sign in when you are ready to upgrade. Stripe Checkout opens in a secure flow and returns here for live status updates.
            </p>
          </div>

          <div data-testid="pricing-summary-card" className="border border-neutral-900 bg-neutral-900 p-8 text-white md:col-span-5">
            <p className="font-mono text-xs uppercase tracking-[0.25em] text-neutral-400">Launch note</p>
            <h2 className="mt-4 text-3xl font-semibold tracking-tight">Every plan keeps the product simple to understand on day one.</h2>
            <p className="mt-4 text-sm leading-relaxed text-neutral-300">
              Start with a free account, then move into a paid workflow once you want stronger watcher coverage and a cleaner review process.
            </p>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-14">
        {statusMessage && (
          <div data-testid="pricing-status-banner" className={`mb-8 border p-4 text-sm font-mono ${statusClass}`}>
            {statusMessage}
          </div>
        )}

        <SectionHeader title="Pricing" meta="MONTHLY PLANS" accentColor="blue" testId="pricing-section-header" />

        <div className="grid gap-6 lg:grid-cols-3">
          <BentoCard testId="pricing-free-card" className="flex h-full flex-col justify-between border-dashed bg-white p-8">
            <div>
              <p className="font-mono text-xs uppercase tracking-[0.25em] text-neutral-500">Free</p>
              <h2 className="mt-4 text-3xl font-semibold text-neutral-900">$0</h2>
              <p className="mt-4 text-sm leading-relaxed text-neutral-600">
                Create your account, explore the dashboard shell, and validate the product before activating a paid plan.
              </p>
            </div>
            <Button
              testId="pricing-free-cta-button"
              variant="secondary"
              className="mt-8 rounded-none"
              onClick={() => router.push(user ? "/dashboard" : "/auth/register")}
            >
              {user ? "Go to dashboard" : "Create free account"}
            </Button>
          </BentoCard>

          {plansLoading
            ? Array.from({ length: 2 }).map((_, index) => (
                <div key={index} className="h-80 animate-pulse border border-neutral-200 bg-white" />
              ))
            : plans.map((plan, index) => {
                const highlighted = index === 0;

                return (
                  <BentoCard
                    key={plan.id}
                    testId={`pricing-plan-card-${plan.id}`}
                    className={`flex h-full flex-col justify-between p-8 ${highlighted ? "border-2 border-neutral-900" : ""}`}
                  >
                    <div>
                      <div className="flex items-center justify-between gap-4">
                        <div>
                          <p className="font-mono text-xs uppercase tracking-[0.25em] text-blue-700">{plan.name}</p>
                          <h2 className="mt-4 text-4xl font-semibold tracking-tight text-neutral-900">
                            {plan.amount_display}
                            <span className="ml-2 text-sm font-mono text-neutral-500">/{plan.interval}</span>
                          </h2>
                        </div>
                        {highlighted && (
                          <span data-testid="pricing-most-popular-badge" className="border border-neutral-900 px-3 py-1 font-mono text-xs uppercase tracking-[0.2em] text-neutral-900">
                            Most Popular
                          </span>
                        )}
                      </div>
                      <p className="mt-5 text-sm leading-relaxed text-neutral-600">{plan.description}</p>
                      <ul className="mt-8 space-y-3">
                        {plan.features.map((feature) => (
                          <li key={feature} className="flex items-start gap-3 text-sm text-neutral-700">
                            <span className="mt-0.5 h-2.5 w-2.5 shrink-0 border border-neutral-900 bg-neutral-900" />
                            <span>{feature}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                    <Button
                      testId={`pricing-checkout-button-${plan.id}`}
                      className="mt-8 rounded-none"
                      onClick={() => handleCheckout(plan.id)}
                      loading={checkoutPlanId === plan.id}
                      loadingText="Opening checkout"
                    >
                      {user ? `Choose ${plan.name}` : "Create account to upgrade"}
                    </Button>
                  </BentoCard>
                );
              })}
        </div>
      </section>

      <section className="border-t border-neutral-200 bg-white px-6 py-16">
        <div className="mx-auto max-w-6xl">
          <SectionHeader title="Questions" meta="FAQ" accentColor="blue" compact testId="pricing-faq-header" />
          <div className="divide-y divide-neutral-200 border-y border-neutral-200">
            {FAQS.map((item) => (
              <div key={item.question} data-testid={`pricing-faq-${item.question.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`} className="grid gap-4 py-6 md:grid-cols-12">
                <h3 className="md:col-span-4 text-base font-semibold text-neutral-900">{item.question}</h3>
                <p className="md:col-span-8 text-sm leading-relaxed text-neutral-600">{item.answer}</p>
              </div>
            ))}
          </div>
        </div>
      </section>
    </main>
  );
}

export default function PricingPage() {
  return (
    <Suspense fallback={<main id="main-content" className="min-h-screen bg-neutral-50" />}>
      <PricingContent />
    </Suspense>
  );
}