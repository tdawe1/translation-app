# WebSocket Client Integration

The GengoWatcher frontend maintains a persistent WebSocket connection to the backend to receive real-time job alerts and system status updates.

## 1. Connection Management

We use a custom hook, `useWatcherWebSocket`, to manage the lifecycle of the connection.

### Features
- **Ticket Request**: Automatically fetches a connection ticket before connecting.
- **Auto-Reconnect**: Implements exponential backoff if the connection is lost.
- **Ping/Pong**: Responds to server heartbeats to prevent connection timeouts.

---

## 2. Event Handling

When a message is received from the WebSocket, the client dispatches it to the appropriate store or query.

| Message Type | Client Action |
|--------------|---------------|
| `JOB_FOUND` | Triggers a toast notification + invalidates `['jobs']` cache. |
| `WATCHER_ERROR`| Triggers a global error alert. |
| `TIER_UPDATED` | Refetches user profile and updates permissions. |

---

## 3. Usage in Components

```tsx
const JobNotificationListener = () => {
  useWatcherWebSocket({
    onJobFound: (job) => {
      toast.success(`New Job: ${job.title}`);
      playSound('success');
    }
  });

  return null; // Invisible listener
};
```

---

## 4. Security

- **SSL**: Connections always use `wss://` in production.
- **Cleanup**: The connection is automatically closed when the user logs out or closes the browser tab.
- **Validation**: Incoming messages are validated against a Zod schema before being processed.

---

## 5. Troubleshooting

### Connection Status
The dashboard includes a "Live" status indicator:
- **Green**: Connected and receiving updates.
- **Yellow**: Attempting to reconnect.
- **Red**: Disconnected/Error.

### Common Issues
- **`403 Forbidden`**: Your subscription tier does not support real-time WebSockets.
- **Ticket Expired**: The connection process took too long; the hook will automatically retry with a new ticket.

## Next Steps
- [Real-Time Notifications Overview](../core-concepts/real-time-notifications.md)
- [WebSocket API Reference](../api/websocket-api.md)
