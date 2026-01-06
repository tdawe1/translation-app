# Setting Up Custom Notifications

GengoWatcher provides multiple channels to alert you of new translation jobs. This guide helps you configure these channels for maximum effectiveness.

## 1. Browser Notifications (Web)

Native browser alerts appear even if you are working in another tab or have the browser minimized.

### Configuration
1. Go to **Dashboard**.
2. Click **"Enable Browser Notifications"**.
3. When prompted by your browser, click **"Allow"**.

**Best For**: Active working sessions on your computer.

---

## 2. Audio Alerts

A distinct sound plays when a job is found.

### Configuration
1. Navigate to **Settings > Notifications**.
2. Toggle **"Sound Alerts"** to ON.
3. Choose from a variety of sounds (e.g., "Sonar", "Success", "Classic Bell").
4. Adjust the volume slider.

**Best For**: Listening for jobs while doing other tasks around the house/office.

---

## 3. Email Notifications

Receive job summaries directly in your inbox via **Resend**.

### Configuration
1. Go to **Settings > Notifications**.
2. Enter your **Notification Email** (can be different from your login email).
3. Select your frequency:
   - **Instant**: One email per job.
   - **Digest**: A summary every 15, 30, or 60 minutes.
   - **None**: Turn off emails.

**Best For**: Staying updated while on mobile or away from your desk.

---

## 4. Mobile Push (via Email/PWA)

While we don't currently have a native mobile app, you can get mobile alerts by:

### Option A: Email Push
Enable **Instant Email Notifications**. Most modern smartphones will show a push notification for new emails.

### Option B: PWA (Progressive Web App)
1. Open GengoWatcher in Chrome (Android) or Safari (iOS).
2. Tap **"Add to Home Screen"**.
3. Open the app from your home screen.
4. Enable browser notifications within the PWA.

---

## Channel Prioritization Strategy

We recommend the following setup for serious translators:

| Scenario | Recommended Setup |
|----------|-------------------|
| **At Desk** | WebSockets + Sound + Browser Notifs |
| **Away (Mobile)** | Instant Email |
| **Sleeping** | All OFF (except High-Reward Email) |

## Next Steps
- [Real-Time Notifications Overview](../core-concepts/real-time-notifications.md)
- [Watcher Configuration](../core-concepts/watcher-system.md)
