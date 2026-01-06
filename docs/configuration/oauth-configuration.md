# OAuth Configuration

GengoWatcher supports **Google** and **GitHub** for social authentication. This guide covers how to set up these providers.

## 1. Google OAuth Setup

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Create a new project or select an existing one.
3. Navigate to **APIs & Services > Credentials**.
4. Click **Create Credentials > OAuth client ID**.
5. **Application Type**: Web application.
6. **Authorized Redirect URIs**:
   - `https://api.gengowatcher.com/api/v1/oauth/google/callback`
7. Copy the **Client ID** and **Client Secret**.

---

## 2. GitHub OAuth Setup

1. Log in to your GitHub account and go to **Settings > Developer settings**.
2. Select **OAuth Apps** and click **New OAuth App**.
3. **Application name**: GengoWatcher.
4. **Homepage URL**: `https://gengowatcher.com`.
5. **Authorization callback URL**:
   - `https://api.gengowatcher.com/api/v1/oauth/github/callback`
6. Click **Register application**.
7. Generate a new **Client Secret**.
8. Copy the **Client ID** and **Client Secret**.

---

## 3. Backend Configuration

Add the credentials to your `backend/.env` file:

```bash
# Google
GOOGLE_OAUTH_CLIENT_ID=your_google_id
GOOGLE_OAUTH_CLIENT_SECRET=your_google_secret

# GitHub
GITHUB_OAUTH_CLIENT_ID=your_github_id
GITHUB_OAUTH_CLIENT_SECRET=your_github_secret

# General
OAUTH_REDIRECT_URL=https://app.gengowatcher.com/auth/callback
```

---

## 4. Security Considerations

- **State Parameter**: GengoWatcher automatically handles the `state` parameter to prevent CSRF attacks during the OAuth flow.
- **Scope**: We only request minimal scopes:
  - Google: `openid`, `profile`, `email`
  - GitHub: `user:email`

---

## 5. Troubleshooting

### "Redirect URI mismatch"
Ensure the URL in the Google/GitHub console exactly matches your `OAUTH_REDIRECT_URL`. Note that `http` vs `https` and trailing slashes matter.

### "Invalid State"
This occurs if the OAuth flow takes too long (timeout > 10 minutes) or if the user's session cookies are blocked.

## Next Steps
- [Multi-Provider Auth Guide](../guides/multi-provider-auth.md)
- [OAuth API Reference](../api/oauth-endpoints.md)
