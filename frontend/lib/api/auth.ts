/**
 * Authentication API endpoints
 */

import { client } from "./client";
import type {
  AuthResponse,
  RegisterRequest,
  LoginRequest,
  ChangePasswordRequest,
  User,
} from "./types";

export const authApi = {
  register: (data: RegisterRequest): Promise<AuthResponse> =>
    client.post<AuthResponse>("/api/v1/auth/register", data),

  login: (data: LoginRequest): Promise<AuthResponse> =>
    client.post<AuthResponse>("/api/v1/auth/login", data),

  logout: (): Promise<void> =>
    client.post<void>("/api/v1/auth/logout"),

  me: (): Promise<User | null> => client.get<User>("/api/v1/me", { optional: true }),

  changePassword: (data: ChangePasswordRequest): Promise<{ message: string }> =>
    client.put<{ message: string }>("/api/v1/me/password", data),

  getWSTicket: (): Promise<{ ticket: string; expires_at: number }> =>
    client.post<{ ticket: string; expires_at: number }>("/api/v1/auth/ws-ticket"),
};
