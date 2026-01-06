# Authentication Guide

GengoWatcher SaaS provides a flexible and secure authentication system supporting multiple methods to accommodate different user preferences and security requirements.

## Supported Methods

| Method | Best For | Security Level |
|--------|----------|----------------|
| **Email/Password** | Traditional users | High (Bcrypt) |
| **Magic Link** | Mobile/Passwordless users | High (One-time use) |
| **OAuth (Google/GitHub)** | Quick onboarding | Highest (Provider-managed) |
| **API Keys** | Programmatic integrations | High (Encrypted storage) |

---

## 1. Email and Password
Users can register with a unique email address and a secure password.

### Password Requirements
- Minimum 8 characters
- Hashed using **Argon2id** (server-side)
- Encouraged to use a mix of characters

### Registration Flow
1. User submits email and password.
2. System creates a pending account.
3. Verification email is sent via **Resend**.
4. User clicks verification link to activate account.

---

## 2. Magic Link (Passwordless)
A secure way to log in without remembering a password.

### How it Works
1. User enters their email on the Login page.
2. A unique, short-lived token (15-minute expiry) is generated.
3. A link is sent to the user's inbox.
4. Clicking the link validates the token and establishes a session.

---

## 3. OAuth 2.0 (Social Login)
One-click login using trusted providers.

### Google
- Recommended for individual freelancers.
- Fast onboarding without manual email verification.

### GitHub
- Recommended for developers and technical users.

---

## 4. Session Management

### JWT (JSON Web Tokens)
We use a dual-token system for security and performance:
- **Access Token**: Short-lived (15 minutes), used for API requests.
- **Refresh Token**: Long-lived (7 days), stored in an `httpOnly`, `Secure`, `SameSite=Strict` cookie.

### Logout
Logging out clears the session on the client and invalidates the refresh token in the database, ensuring the session cannot be resumed.

---

## 5. Security Features

### Rate Limiting
To prevent brute-force attacks, we limit authentication attempts:
- **Login/Register**: 5 attempts per minute per IP.
- **Magic Link/Password Reset**: 3 requests per hour per email.

### CSRF Protection
All state-changing requests require a valid session and use secure cookie attributes to prevent Cross-Site Request Forgery.

### Multi-Tenant Isolation
Authenticated users are strictly isolated. All data queries are scoped by the `user_id` extracted from the JWT, ensuring users can only see and manage their own watchers.

---

## Troubleshooting

### Verification Email Not Received
- Check your Spam/Junk folder.
- Ensure the email address was typed correctly.
- Wait 5 minutes; delivery can sometimes be delayed.

### "Token Expired"
Magic links and password resets expire quickly for security. Please request a new link if yours has expired.

## Next Steps
- [API Reference](../api/overview.md)
- [Watcher Configuration](../core-concepts/watcher-system.md)
