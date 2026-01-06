# WebSocket API

GengoWatcher SaaS uses WebSockets to push real-time job notifications directly to your application or browser.

## Connection Lifecycle

### 1. Obtain a Connection Ticket
To prevent unauthorized access, we use a short-lived ticket system.

- **URL**: `/api/v1/auth/ws-ticket`
- **Method**: `POST`
- **Auth Required**: Yes

**Success Response (200):**
```json
{
  "data": {
    "ticket": "ws_tk_abc123xyz...",
    "expires_at": "2026-01-05T23:15:00Z"
  }
}
```

### 2. Connect
Connect to the WebSocket endpoint using the ticket in the query string.

- **URL**: `wss://api.gengowatcher.com/ws?ticket=<your_ticket>`

---

## Messages (Server to Client)

The server sends messages as JSON objects with a `type` and `payload`.

### New Job Found
Sent when a watcher discovers a job matching your criteria.
```json
{
  "type": "JOB_FOUND",
  "payload": {
    "id": "job_99",
    "title": "Document Translation",
    "reward": 22.50,
    "language_pair": "en-de",
    "url": "https://gengo.com/jobs/99"
  }
}
```

### Watcher Error
Sent when your background watcher encounters an operational error.
```json
{
  "type": "WATCHER_ERROR",
  "payload": {
    "code": "RSS_FETCH_FAILED",
    "message": "Unable to reach RSS feed. Please check your URL."
  }
}
```

### Heartbeat
Sent every 30 seconds to keep the connection alive.
```json
{
  "type": "PING",
  "payload": {
    "server_time": "2026-01-05T23:10:00Z"
  }
}
```

---

## Messages (Client to Server)

Currently, the WebSocket connection is primarily for **Server Push**. Actions like starting/stopping watchers should be performed via the [REST API](../api/watcher-endpoints.md).

### Pong
It is recommended to respond to `PING` messages to ensure the connection isn't closed by load balancers.
```json
{
  "type": "PONG"
}
```

---

## Connection Limits
- **Maximum Connections**: 1 per user (Free), 5 per user (Pro/Enterprise).
- **Idle Timeout**: 5 minutes (if no heartbeat).
- **Session Duration**: Connections are closed after 24 hours and must be re-established.

## Implementation Example (JavaScript)

```javascript
const connectWS = async () => {
  // 1. Get ticket
  const res = await fetch('/api/v1/auth/ws-ticket', { method: 'POST' });
  const { data } = await res.json();

  // 2. Connect
  const socket = new WebSocket(`wss://api.gengowatcher.com/ws?ticket=${data.ticket}`);

  socket.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.type === 'JOB_FOUND') {
      alert(`New job: ${msg.payload.title}`);
    }
  };
};
```

## Next Steps
- [Real-Time Notifications Overview](../core-concepts/real-time-notifications.md)
- [Watcher Endpoints](../api/watcher-endpoints.md)
