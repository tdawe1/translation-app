# AGENTS.md - Next.js Frontend

> **Parent**: [../AGENTS.md](../AGENTS.md)

## Overview

Next.js 16 application using App Router and React 19. Implements "Data Factory" design language with Zustand state management.

## Quick Commands

```bash
npm run dev              # Development server (Turbopack)
npm run build            # Production build
npm run test             # Run Vitest tests
npm run test:ui          # Vitest UI
npm run test:coverage    # Coverage report
npm run lint             # ESLint
```

## Directory Structure

```
frontend/
├── app/                     # App Router routes
│   ├── layout.tsx           # Root layout (providers, fonts)
│   ├── page.tsx             # Home page (/)
│   ├── auth/                # Auth routes (login, register, magic-link)
│   ├── dashboard/           # Main dashboard
│   └── settings/            # User settings
├── components/              # React components
│   ├── auth/                # Auth components (4 files)
│   ├── watcher/             # Watcher components (5 files)
│   ├── realtime/            # WebSocket components (5 files)
│   └── ui/                  # UI kit
│       ├── base/            # Data Factory components (4 files)
│       └── states/          # Loading/error states (4 files)
├── store/                   # Zustand stores (5 files)
│   ├── useAuthStore.ts      # User session
│   ├── useWatcherStore.ts   # Watcher config
│   ├── useRealtimeStore.ts  # WebSocket
│   ├── useJobsStore.ts      # Jobs list
│   └── useToastStore.ts     # Notifications
├── lib/
│   ├── api/                 # API client (6 files)
│   │   ├── client.ts        # HttpClient with interceptors
│   │   ├── auth.ts          # Auth endpoints
│   │   └── watcher.ts       # Watcher endpoints
│   └── design/
│       └── tokens.ts        # Design system tokens
└── tests/
    └── smoke/               # Smoke tests (6 files)
```

## Complexity Hotspot

| File | Lines | Issue |
|------|-------|-------|
| `job-filter-panel.tsx` | 554 | Mega-component: data + UI + logic in one file |

**Recommendation**: Extract `filterJobs` function and filter presets to separate files.

## Design Language: "Data Factory"

- **Fonts**: IBM Plex Sans (headings), IBM Plex Mono (labels)
- **Cards**: Bento style, 1px border, sharp corners
- **Hover**: Precision focus (border color, NO shadow lift)
- **Accents**: ROYGBIV spectrum for headings ONLY
- **Spacing**: Generous (py-24 to pt-44)

## Code Patterns

### Zustand Store Pattern
```typescript
import { create } from "zustand";
import { persist } from "zustand/middleware";

interface MyStoreState {
  data: DataType | null;
  loading: boolean;
  fetchData: () => Promise<void>;
}

export const useMyStore = create<MyStoreState>()(
  persist(
    (set) => ({
      data: null,
      loading: false,
      fetchData: async () => {
        set({ loading: true });
        const data = await apiClient.getData();
        set({ data, loading: false });
      },
    }),
    {
      name: "gengowatcher-mystore",
      partialize: (state) => ({ data: state.data }),
    }
  )
);
```

### API Client Pattern
```typescript
import { authApi, watcherApi } from "@/lib/api";

// Auth
await authApi.register(email, password);
await authApi.login(email, password);
await authApi.me();

// Watcher
await watcherApi.getConfig();
await watcherApi.start();
```

### Component Structure
```typescript
// Domain components in components/{domain}/
// UI primitives in components/ui/base/
// Use BentoCard for card layouts
import { BentoCard } from "@/components/ui/base/BentoCard";
```

## Route Conventions

| Route | Access | Purpose |
|-------|--------|---------|
| `/` | Public | Landing page |
| `/auth/*` | Public | Login, register, magic-link |
| `/dashboard/*` | Protected | Main app |
| `/settings/*` | Protected | User settings |

## MUST NOT

- Use shadow effects on hover (precision focus only)
- Mix domain logic with UI components
- Skip TypeScript strict mode
- Use inline styles (use Tailwind)
- Duplicate API calls (HttpClient deduplicates)

## Testing Pattern

```typescript
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';

describe('MyComponent', () => {
  it('renders correctly', () => {
    render(<MyComponent user={{ email: 'test@example.com' }} />);
    expect(screen.getByText('test@example.com')).toBeInTheDocument();
  });
});
```

---

*For full development guide, see CLAUDE.md at project root*
