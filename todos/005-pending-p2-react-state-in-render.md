---
status: resolved
priority: p2
issue_id: "005"
tags:
  - frontend
  - react
  - bug
  - performance
  - code-review
dependencies: []
---

# P2: React State Update in Render Body Anti-Pattern

## Problem Statement

State update is triggered directly in the render body, which can cause infinite loops or unexpected re-renders.

**File**: `frontend/components/watcher/config-form.tsx:35-37`

```typescript
if (config && JSON.stringify(formData) !== JSON.stringify(config)) {
  setFormData(config);
}
```

## Findings

### Why This Is Wrong

1. **Causes Extra Render**: State update triggers re-render
2. **Unreliable Comparison**: `JSON.stringify` key ordering affects equality
3. **Anti-Pattern**: State updates should be in event handlers or `useEffect`
4. **Potential Infinite Loop**: If new state triggers same condition

### Impact
- **Severity**: IMPORTANT - Causes performance issues and potential infinite loops
- **Confidence**: 85/100 - Clear React anti-pattern

### Evidence
- Location: `components/watcher/config-form.tsx:35-37`
- Runs on every render
- Uses brittle `JSON.stringify` comparison

## Proposed Solutions

### Option 1: useEffect with Dependency Array (Recommended)

```typescript
useEffect(() => {
  if (config) {
    setFormData(config);
  }
}, [config]);
```

**Pros**:
- Standard React pattern
- Only runs when `config` actually changes
- No risk of infinite loop
- Uses React's dependency tracking

**Cons**:
- Requires adding `useEffect` import
- Slightly more verbose

**Effort**: Small
**Risk**: Low

### Option 2: Deep Equality Check

```typescript
import { isEqual } from 'lodash';

// In render
if (config && !isEqual(formData, config)) {
  setFormData(config);
}
```

**Pros**:
- More reliable comparison than JSON.stringify
- Handles nested objects properly

**Cons**:
- Still in render body (anti-pattern)
- Adds lodash dependency
- Can still cause issues

**Effort**: Small
**Risk**: Medium (still anti-pattern)

### Option 3: useSyncExternalStore (Advanced)

```typescript
const formData = useSyncExternalStore(
  () => store.getState().config,
  () => store.getState().config
);
```

**Pros**:
- No local state needed
- Always in sync with store
- Most efficient

**Cons**:
- Can't modify form data locally before submit
- Different mental model
- More complex for form handling

**Effort**: Medium
**Risk**: Medium

## Recommended Action

**Implement Option 1** - Use `useEffect` with proper dependency array.

## Technical Details

### Affected Files
- `frontend/components/watcher/config-form.tsx:35-37`

### Components
- Config form component
- Form state synchronization

### Acceptance Criteria

- [ ] State update moved to `useEffect`
- [ ] Dependency array includes `config`
- [ ] No more direct state updates in render
- [ ] Form still syncs with config changes
- [ ] No infinite loops in React DevTools
- [ ] Tests verify form state updates correctly

## Work Log

### 2025-12-29
- **Finding**: Frontend review identified state update in render
- **Analysis**: Confirmed anti-pattern with potential for infinite loop
- **Decision**: Selected useEffect approach
- **Status**: Pending implementation
