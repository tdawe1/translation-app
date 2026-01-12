/**
 * OAuth API endpoints
 */

import { client } from "./client";
import type { OAuthAccount } from "./types";

export const oauthApi = {
  authorize: (provider: "google" | "github"): Promise<{ auth_url: string }> =>
    client.get<{ auth_url: string }>(`/api/v1/oauth/authorize?provider=${provider}`),

  getLinkedAccounts: (): Promise<{ linked_accounts: OAuthAccount[] }> =>
    client.get<{ linked_accounts: OAuthAccount[] }>("/api/v1/oauth/accounts"),

  unlinkAccount: (provider: "google" | "github"): Promise<void> =>
    client.delete<void>(`/api/v1/oauth/${provider}`),
};
