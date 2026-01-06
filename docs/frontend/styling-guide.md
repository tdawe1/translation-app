# Styling & Design Guide

GengoWatcher SaaS follows a "Data Factory" design language: professional, precise, and high-performance.

## 1. Core Technologies
- **Tailwind CSS 4.x**: For all styling.
- **Radix UI**: For accessible, unstyled UI primitives.
- **IBM Plex Sans**: Primary font for headings and body.
- **IBM Plex Mono**: For labels, IDs, and reward amounts.

---

## 2. Design System

### Colors
We use a neutral palette with vibrant accents for status indicators.

- **Background**: `bg-zinc-50` (Primary), `bg-white` (Cards).
- **Text**: `text-zinc-900` (Body), `text-zinc-500` (Secondary).
- **Accents**:
  - `emerald`: Success / Running / Low-Reward.
  - `amber`: Warning / Medium-Reward.
  - `rose`: Error / Stopped / High-Reward.

### Typography
- **Headings**: Semibold, IBM Plex Sans.
- **Data**: Medium, IBM Plex Mono (monospaced for alignment).
- **Labels**: Uppercase, tracked, 10px-12px.

---

## 3. Layout Patterns

### Bento Dashboard
Information is divided into clear, logical sections.
- **Gaps**: `gap-4` or `gap-6`.
- **Borders**: `border border-zinc-200`.
- **Shadows**: None (we use 1px borders instead for a "flat" professional look).

### Sections
Generous vertical padding between sections:
- `py-24` (desktop) to `py-12` (mobile).

---

## 4. Components Style

### Forms
- **Corners**: Sharp (no `rounded` classes).
- **Focus**: `ring-2 ring-zinc-900 ring-offset-2`.

### Tables
- **Stripes**: `even:bg-zinc-50`.
- **Headers**: Sticky, semi-transparent background.

---

## 5. Responsive Design

We follow a strict mobile-first approach:
1. **Mobile (<640px)**: Single column layouts, hidden sidebars.
2. **Tablet (640px - 1024px)**: Grid layouts (2 columns).
3. **Desktop (>1024px)**: Full multi-column dashboard with persistent sidebar.

---

## 6. Utilities & Extensions

Custom Tailwind utilities are defined in `frontend/tailwind.config.js`. Avoid writing custom CSS in `.css` files unless absolutely necessary for third-party library overrides.

## Next Steps
- [Component Library](../frontend/component-library.md)
- [Frontend Overview](../docs/README.md)
