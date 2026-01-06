# Frontend Component Library

GengoWatcher SaaS uses a modern, utility-first component architecture based on **React 19**, **Tailwind CSS**, and **Lucide Icons**.

## 1. Design Principles

- **Bento Style**: Information is organized into distinct, clean cards with 1px borders.
- **Precision Focus**: Sharp corners (no rounded edges) and high-contrast borders.
- **Responsive**: All components are designed with a mobile-first approach.
- **Accessibility**: ARIA labels and keyboard navigation are prioritized.

---

## 2. Core Components

### `Card`
The fundamental building block for the dashboard.
```tsx
import { Card } from '@/components/ui/card';

<Card title="Watcher Status">
  <div className="p-4">Running</div>
</Card>
```

### `Button`
Standardized buttons for actions.
- **Variants**: `primary`, `secondary`, `danger`, `outline`.
- **States**: `loading`, `disabled`, `hover`.

### `Badge`
Used for statuses and language pairs.
- **Colors**: Green (Success/Running), Red (Error/Stopped), Blue (Info).

---

## 3. UI Patterns

### Forms
We use **React Hook Form** with **Zod** for schema validation.
- Inputs are styled with consistent borders and focus states.
- Error messages appear immediately below the field.

### Data Tables
Used for job history.
- Support for sorting and pagination.
- Row-level actions (e.g., "Ignore Job", "Open in Gengo").

### Notifications (Toast)
Real-time feedback for user actions (e.g., "Settings Saved") and job discoveries.

---

## 4. Icons
We use **Lucide React** for consistent, lightweight vector icons.
- `Play` / `Square`: For watcher controls.
- `Settings`: For user configuration.
- `Bell`: For notification status.

---

## 5. Development

### Previewing Components
We recommend using **Storybook** (if configured) or the `/dev/styleguide` route to preview and test components in isolation.

### Creating New Components
1. Place common UI elements in `frontend/components/ui/`.
2. Place feature-specific components (e.g., `WatcherConfigForm`) in `frontend/components/watcher/`.
3. Use Tailwind classes directly for styling; avoid custom CSS files where possible.

## Next Steps
- [Styling Guide](../frontend/styling-guide.md)
- [State Management](../frontend/state-management.md)
