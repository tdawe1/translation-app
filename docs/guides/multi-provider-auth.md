# Using Multiple OAuth Providers

GengoWatcher allows you to link multiple social accounts (Google and GitHub) to a single user profile. this provides flexibility and ensures you never lose access to your account.

## Benefits
- **Redundancy**: Log in with either Google or GitHub.
- **Flexibility**: Use your work email via Google and your personal email via GitHub.
- **Security**: If one provider is down, you can still access GengoWatcher.

---

## Linking a Second Provider

If you already have an account (e.g., created via Email/Password or Google), you can link another provider:

1. Log in to your GengoWatcher account.
2. Navigate to **Settings > Security**.
3. Under **Social Accounts**, you will see available providers.
4. Click **"Link GitHub Account"** (or Google).
5. Complete the OAuth flow with the provider.

The accounts are now linked. You can henceforth log in with either method.

---

## Account Merging Logic

What happens if you have an existing account with `user@example.com` and then try to "Login with Google" using that same email?

1. **Email Match**: If the emails match exactly, GengoWatcher will automatically link the Google account to your existing profile after successful authentication.
2. **Security Check**: For your protection, we may require you to enter your GengoWatcher password once to verify the link if you haven't verified your email yet.

---

## Unlinking a Provider

You can remove a social connection at any time:

1. Go to **Settings > Security**.
2. Click **"Unlink"** next to the provider.

**⚠️ Restriction**: You cannot unlink a provider if it is your *only* way to log in. You must have at least one other linked OAuth account or an Email/Password combination set up.

---

## Troubleshooting

### "Email already in use"
If you try to link a GitHub account that uses an email already associated with a *different* GengoWatcher user, the link will fail. You must log in to that other account and delete it, or change its email address first.

### Missing Profile Data
If a provider doesn't share your email address (common with some GitHub privacy settings), you may be prompted to enter it manually before the account can be created or linked.

## Next Steps
- [Authentication Overview](../getting-started/authentication.md)
- [OAuth API Reference](../api/oauth-endpoints.md)
