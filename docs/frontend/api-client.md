# Using the API Client

The GengoWatcher frontend communicates with the backend via a standardized API client based on **Axios**.

## 1. Setup

The client is configured in `frontend/lib/api.ts`. It includes:
- **`baseURL`**: Pointing to your `NEXT_PUBLIC_API_URL`.
- **`withCredentials: true`**: Ensures that cookies are sent with every request.
- **Request Interceptors**: Injects the JWT access token into the `Authorization` header.

---

## 2. Basic Usage

```typescript
import { api } from '@/lib/api';

// GET request
const fetchConfig = async () => {
  const { data } = await api.get('/watcher/config');
  return data.data;
};

// POST request
const startWatcher = async () => {
  await api.post('/watcher/start');
};
```

---

## 3. Integrating with TanStack Query

For most UI components, you should wrap API calls in a query hook:

```typescript
import { useQuery } from '@tanstack/react-query';

export const useWatcherConfig = () => {
  return useQuery({
    queryKey: ['watcher-config'],
    queryFn: async () => {
      const { data } = await api.get('/watcher/config');
      return data.data;
    },
  });
};
```

---

## 4. Error Handling

We use an **Axios Interceptor** to catch errors globally.
- **401**: Triggers the session refresh flow.
- **429**: Shows a "Rate Limited" toast notification.
- **500**: Shows a "System Error" toast notification.

---

## 5. Token Refresh Logic

The client automatically handles token expiration:
1. A request fails with `401`.
2. The client pauses further requests.
3. It calls `/auth/refresh` to get a new JWT.
4. If successful, it retries the original request.
5. If it fails, it clears the auth store and redirects to login.

---

## 6. Type Safety

We use TypeScript interfaces to define request and response shapes, ensuring type safety throughout the frontend.

```typescript
interface WatcherConfig {
  min_reward: number;
  rss_feed_url: string;
}

const { data } = await api.get<ApiResponse<WatcherConfig>>('/watcher/config');
```

## Next Steps
- [State Management](../frontend/state-management.md)
- [API Overview (Reference)](../api/overview.md)
