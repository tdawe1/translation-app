# Real-Time Notifications

GengoWatcher SaaS is designed for speed. When a translation job is posted, every second counts. Our real-time notification system ensures you receive alerts as fast as the internet allows.

## The Technology Stack

We use a combination of three technologies to achieve low-latency notifications:

1. **WebSockets**: For the persistent connection between your browser and our server.
2. **Redis Pub/Sub**: For internal communication between background watcher processes and the API server.
3. **Browser Notifications**: For native OS alerts even when the tab is in the background.

---

## The Workflow

### 1. Job Discovery
A background "Watcher" routine (running in Go) identifies a new job on a translation platform.

### 2. Internal Messaging (Redis)
If the job matches your filters, the backend publishes a JSON message to a Redis channel dedicated to you:
- **Channel**: `user:{user_id}:jobs`
- **Payload**:
  ```json
  {
    "id": "job_123",
    "title": "English to Spanish Translation",
    "reward": 15.50,
    "url": "https://gengo.com/jobs/123"
  }
  ```

### 3. Dispatch (WebSockets)
Our WebSocket handler is subscribed to your Redis channel. As soon as the message arrives in Redis, the server pushes it down your open WebSocket connection.

### 4. Client-Side Alert
Your browser receives the message and:
- Updates the **Live Jobs** list on your dashboard.
- Plays a notification sound (if enabled).
- Triggers a browser desktop notification.

---

## Connection Management

### WebSocket Tickets
For security, we don't allow direct connection to WebSockets using only cookies or long-lived tokens.
1. The client requests a short-lived **WebSocket Ticket** via a POST request.
2. The server generates a random string and stores it in Redis with a 60-second expiry.
3. The client connects to `/ws?ticket=YOUR_TICKET`.
4. The server validates and deletes the ticket upon successful connection.

### Heartbeats & Reconnection
- **Heartbeats**: The server sends a `ping` every 30 seconds to keep the connection alive.
- **Auto-Reconnection**: If the connection drops, our frontend client uses an exponential backoff strategy to reconnect automatically.

---

## Fallback: Email Notifications
If your WebSocket is disconnected (e.g., you closed your laptop), GengoWatcher can send email alerts via **Resend**.

- **Configurable**: Choose to receive emails for all matches, only high-reward jobs, or none at all.
- **Batching**: To avoid flooding your inbox, you can enable "Digest Mode" to receive a summary every 15 minutes.

## Next Steps
- [Watcher System](../core-concepts/watcher-system.md)
- [WebSocket API Reference](../api/websocket-api.md)
