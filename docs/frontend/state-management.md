# State Management

GengoWatcher SaaS uses a decentralized state management strategy, combining **Zustand** for global UI state and **TanStack Query** for server-state synchronization.

## 1. Global UI State (Zustand)

Zustand provides a lightweight, performant store for data that needs to be accessed across many components.

### Stores
- **`useAuthStore`**: Tracks current user profile and authentication status.
- **`useUIStore`**: Manages sidebar state, theme settings, and toast notifications.
- **`useWatcherStore`**: Tracks the local state of the user's active watcher (running/stopped).

**Example Usage:**
```typescript
import { useAuthStore } from '@/store/auth';

const user = useAuthStore((state) => state.user);
const logout = useAuthStore((state) => state.logout);
```

---

## 2. Server State (TanStack Query)

We use **TanStack Query (v5)** for all data fetching, caching, and synchronization with the Backend API.

### Benefits
- **Automatic Caching**: Data is cached and reused across pages.
- **Background Refetching**: Keeps the dashboard fresh without full page reloads.
- **Loading/Error States**: Simplifies UI logic for async operations.

**Example Hook:**
```typescript
const { data, isLoading } = useQuery({
  queryKey: ['watcher-config'],
  queryFn: fetchWatcherConfig,
});
```

---

## 3. Real-Time State (WebSockets)

WebSocket messages are integrated into our state management:
1. When a `JOB_FOUND` message arrives, the WebSocket listener triggers a query invalidation for `['jobs']`.
2. This causes TanStack Query to refetch the latest jobs list from the API.
3. The UI updates automatically to show the new job at the top of the list.

---

## 4. Local Persistence

We use `zustand/middleware` with **LocalStorage** to persist certain UI preferences (like the preferred language or sidebar position) across browser sessions.

---

## 5. Best Practices
- **Minimize Global State**: Only put data in Zustand if it's truly needed by multiple, distant components.
- **Use Selectors**: Always use selectors to prevent unnecessary re-renders (e.g., `state => state.property`).
- **Query Keys**: Use consistent, hierarchical query keys (e.g., `['user', userId, 'config']`) to allow for targeted cache invalidation.

## Next Steps
- [API Client Guide](../frontend/api-client.md)
- [WebSocket Client Guide](../frontend/websocket-client.md)
