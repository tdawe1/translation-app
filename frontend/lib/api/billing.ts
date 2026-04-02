import { client } from "./client";
import type { BillingCheckoutResponse, BillingPlan, BillingStatusResponse } from "./types";

export const billingApi = {
  getPlans: async (): Promise<BillingPlan[]> => {
    const response = await client.get<{ plans: BillingPlan[] }>("/api/v1/billing/plans");
    return response.plans;
  },

  createCheckout: (planId: string, userEmail?: string | null): Promise<BillingCheckoutResponse> =>
    client.post<BillingCheckoutResponse>("/api/v1/billing/checkout", {
      plan_id: planId,
      origin_url: typeof window !== "undefined" ? window.location.origin : "",
      user_email: userEmail || undefined,
    }),

  getStatus: (sessionId: string): Promise<BillingStatusResponse> =>
    client.get<BillingStatusResponse>(`/api/v1/billing/status/${sessionId}`),
};