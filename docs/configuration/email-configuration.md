# Email Configuration (Resend)

GengoWatcher SaaS uses **Resend** for all transactional emails, including verification, password resets, and job notifications.

## 1. Prerequisites
1. Create an account at [Resend.com](https://resend.com).
2. Verify your sending domain (e.g., `mail.gengowatcher.com`).
3. Generate an API Key.

---

## 2. Environment Variables

Configure the following variables in your `backend/.env`:

| Variable | Description | Example |
|----------|-------------|---------|
| `RESEND_API_KEY` | Your Resend API key. | `re_123456789` |
| `FROM_EMAIL` | The verified sender address. | `no-reply@gengowatcher.com` |
| `FROM_NAME` | The name shown in the "From" field. | `GengoWatcher` |

---

## 3. Supported Email Types

GengoWatcher sends the following types of emails:

### A. Verification Email
Sent upon registration. Includes a link to verify the user's email address.

### B. Magic Link
Sent when a user requests a passwordless login.

### C. Password Reset
Sent when a user requests to reset their password.

### D. Job Alerts
Sent when a new job matching the user's criteria is discovered (if email notifications are enabled).

---

## 4. Local Development (MailHog)

During development, we use **MailHog** to capture emails instead of sending them to real addresses.

- **Backend**: Set `RESEND_API_KEY` to an empty string. The backend will default to sending via local SMTP to MailHog.
- **Access**: Open `http://localhost:8025` in your browser to view captured emails.

---

## 5. Troubleshooting

### Emails Not Received
1. **Check API Key**: Verify `RESEND_API_KEY` is correct.
2. **Domain Verification**: Ensure your domain is "Verified" in the Resend dashboard.
3. **Reputation**: Check if your emails are being flagged as spam.
4. **Logs**: Check backend logs for `Resend API Error: ...`.

### Rate Limits
Resend has its own rate limits based on your plan. Ensure your expected volume of job notifications doesn't exceed these limits.

## Next Steps
- [Authentication Guide](../getting-started/authentication.md)
- [Environment Variables Reference](../configuration/environment-variables.md)
