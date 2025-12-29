---
status: resolved
priority: p2
issue_id: "008"
tags:
  - frontend
  - accessibility
  - a11y
  - ui-components
  - code-review
dependencies: []
---

# P2: Missing ARIA Attributes and Accessibility Features

## Problem Statement

Multiple UI components lack proper ARIA attributes, focus management, and keyboard navigation support.

**Files**:
- `frontend/components/ui/modal.tsx:42-56`
- `frontend/app/dashboard/page.tsx:103-112, 217-236`
- `frontend/components/watcher/job-list.tsx:64-68`

## Findings

### WCAG Compliance Issues

1. **Modal**: No `role="dialog"`, `aria-modal`, `aria-labelledby`
2. **Modal**: No focus trap (keyboard users can escape)
3. **Modal**: No explicit close button for keyboard users
4. **Status Indicators**: No `aria-live` for dynamic status changes
5. **Buttons**: Missing `aria-label` where context unclear
6. **Clear Button**: No `aria-label` in job list

### Impact
- **Severity**: IMPORTANT - Affects keyboard and screen reader users
- **WCAG**: 2.1 Level A compliance issues

### Detailed Findings

#### Modal Component (`ui/modal.tsx`)
```typescript
<div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
  {/* Missing: role="dialog", aria-modal, aria-labelledby */}
  <div className="bg-white rounded-lg shadow-xl">
    {/* Missing: explicit close button with aria-label */}
    {onClose && <button onClick={onClose}>✕</button>}
    {/* Missing: focus management */}
  </div>
</div>
```

#### Dashboard Status (`page.tsx`)
```typescript
{/* Status changes need aria-live */}
<p className={statusColor}>{statusText}</p>
{/* Missing: aria-live="polite" or role="status" */}
```

#### Job List Clear Button
```typescript
<button onClick={clearJobs}>
  {/* Missing: aria-label="Clear all jobs" */}
  Clear Jobs
</button>
```

## Proposed Solutions

### Option 1: Implement Accessible Modal Component (Recommended)

```typescript
interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
}

export function Modal({ isOpen, onClose, title, children }: ModalProps) {
  const modalRef = useRef<HTMLDivElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  // Focus management
  useEffect(() => {
    if (isOpen) {
      // Focus close button on open
      closeButtonRef.current?.focus();
      // Trap focus within modal
      const focusableElements = modalRef.current?.querySelectorAll(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      const firstElement = focusableElements?.[0] as HTMLElement;
      const lastElement = focusableElements?.[focusableElements.length - 1] as HTMLElement;

      const handleTab = (e: KeyboardEvent) => {
        if (e.key === 'Tab') {
          if (e.shiftKey) {
            if (document.activeElement === firstElement) {
              e.preventDefault();
              (lastElement as HTMLElement).focus();
            }
          } else {
            if (document.activeElement === lastElement) {
              e.preventDefault();
              (firstElement as HTMLElement).focus();
            }
          }
        }
      };

      document.addEventListener('keydown', handleTab);
      return () => document.removeEventListener('keydown', handleTab);
    }
  }, [isOpen]);

  // Escape key closes modal
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div
      ref={modalRef}
      role="dialog"
      aria-modal="true"
      aria-labelledby={title ? "modal-title" : undefined}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={(e) => {
        // Close on backdrop click, but not on content click
        if (e.target === e.currentTarget) {
          onClose();
        }
      }}
    >
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
        {title && (
          <div className="px-6 py-4 border-b">
            <h2 id="modal-title" className="text-lg font-light">{title}</h2>
          </div>
        )}
        <div className="p-6">
          {children}
        </div>
        <div className="px-6 py-4 border-t flex justify-end">
          <button
            ref={closeButtonRef}
            onClick={onClose}
            className="px-4 py-2"
            aria-label="Close modal"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
```

**Pros**:
- Full keyboard navigation
- Screen reader friendly
- Focus trap implemented
- Escape key support
- ARIA compliant

**Cons**:
- More complex component
- Requires careful focus management

**Effort**: Medium
**Risk**: Low

### Option 2: Use Radix UI Dialog (Quickest)

```bash
npm install @radix-ui/react-dialog
```

```typescript
import * as Dialog from '@radix-ui/react-dialog';

export function Modal({ isOpen, onClose, children }) {
  return (
    <Dialog.Root open={isOpen} onOpenChange={onClose}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50" />
        <Dialog.Content className="fixed ...">
          {children}
          <Dialog.Close asChild>
            <button aria-label="Close">✕</button>
          </Dialog.Close>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
```

**Pros**:
- Fully accessible out of the box
- Well-tested component
- Less custom code

**Cons**:
- Additional dependency
- May need styling customization

**Effort**: Small
**Risk**: Low

### Option 3: Add Minimal ARIA Attributes (Quickest Fix)

For existing modal, just add attributes:

```typescript
<div
  role="dialog"
  aria-modal="true"
  aria-labelledby="modal-title"
  className="..."
>
  <h2 id="modal-title" className="sr-only">{title}</h2>
  {/* ... */}
</div>
```

**Pros**:
- Minimal code changes
- Improves screen reader experience

**Cons**:
- No focus management
- No keyboard handling
- Partial solution

**Effort**: Small
**Risk**: Medium (incomplete)

## Recommended Action

**Implement Option 2** (Radix UI Dialog) for comprehensive accessibility, or **Option 1** for custom implementation.

## Technical Details

### Affected Files
- `frontend/components/ui/modal.tsx` (complete rewrite)
- `frontend/app/dashboard/page.tsx` (add aria-live)
- `frontend/components/watcher/job-list.tsx` (add aria-label)

### Dependencies
- Consider: `@radix-ui/react-dialog` or `@radix-ui/react-popover`

### Acceptance Criteria

- [ ] Modal has `role="dialog"` and `aria-modal="true"`
- [ ] Modal has proper `aria-labelledby` referencing title
- [ ] Close button has `aria-label="Close modal"`
- [ ] Focus trap implemented within modal
- [ ] Escape key closes modal
- [ ] Status indicators use `aria-live="polite"`
- [ ] All icon buttons have descriptive `aria-label`
- [ ] Keyboard navigation works throughout
- [ ] Screen reader testing passes
- [ ] Lighthouse accessibility score >90

## Work Log

### 2025-12-29
- **Finding**: A11y review identified missing ARIA attributes
- **Analysis**: Confirmed WCAG compliance issues
- **Decision**: Selected Radix UI for reliable accessibility
- **Status**: Pending implementation

## Resources

- [WCAG 2.1](https://www.w3.org/WAI/WCAG21/quickref/)
- [ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)
- [Radix UI Accessibility](https://www.radix-ui.com/docs/primitives/docs/primitives/accessibility)
