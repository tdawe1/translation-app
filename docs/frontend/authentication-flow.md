# Frontend Authentication Flow

This guide explains how authentication is handled on the client side of GengoWatcher SaaS.

## 1. Overview
GengoWatcher uses a secure, cookie-based session strategy.
- **Access Token**: A short-lived JWT stored in memory.
- **Refresh Token**: A long-lived token stored in a secure, `httpOnly` cookie.

---

## 2. The Login Flow

1. **Submission**: User submits credentials (or OAuth flow completes).
2. **Response**: The API returns an `access_token` and the browser automatically receives the `refresh_token` cookie.
3. **Store**: The `access_token` and `user` profile are saved in the Zustand `useAuthStore`.
4. **Redirect**: The user is navigated to the `/dashboard`.

---

## 3. Persistent Sessions

When a user refreshes the page:
1. **Initial Load**: The `AuthProvider` component checks if an access token exists.
2. **Silent Refresh**: If not, it calls the `/api/v1/auth/refresh` endpoint.
3. **Cookie Validation**: The browser sends the `refresh_token` cookie; if valid, the API returns a new access token.
4. **Hydration**: The app hydrates the store and renders the protected content.
5. **Fail**: If the cookie is invalid or expired, the user is redirected to `/login`.

---

## 4. Protecting Routes

We use a higher-order component (HOC) or an Next.js **Middleware** to protect routes.

```typescript
// middleware.ts
export function middleware(request: NextRequest) {
  const token = request.cookies.get('refresh_token');
  
  if (!token && isProtectedRoute(request.url)) {
    return NextResponse.redirect(new URL('/login', request.url));
  }
}
```

---

## 5. Handling Session Expiry

If an API request fails with a `401 Unauthorized`:
1. The **API Client** (Axios interceptor) attempts to call the refresh endpoint once.
2. If successful, it retries the original request with the new token.
3. If the refresh also fails, it triggers the `logout` action in Zustand and clears the local state.

---

## 6. OAuth Callback Handling

1. The OAuth provider redirects to `/auth/callback?code=...&state=...`.
2. The page shows a "Authenticating..." loading state.
3. A request is sent to the backend `/oauth/{provider}/callback`.
4. Upon success, the user is redirected to the main dashboard.

## Next Steps
- [Authentication Guide (Core)](../getting-started/authentication.md)
- [API Client Guide](../frontend/api-client.md)
