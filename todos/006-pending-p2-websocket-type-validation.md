---
status: resolved
priority: p2
issue_id: "006"
tags:
  - frontend
  - typescript
  - type-safety
  - websocket
  - code-review
dependencies: []
---

# P2: Unsafe Type Casting in WebSocket Hook

## Problem Statement

The WebSocket hook uses unsafe type casts (`as unknown as Job`) that bypass TypeScript's type checking and could lead to runtime errors.

**File**: `frontend/hooks/use-watcher-websocket.ts:47-48, 132-146`

```typescript
const jobData = "data" in message && typeof message.data === "object"
  ? (message.data as unknown as Job)  // 🔴 Unsafe cast
  : (message as unknown as Job);       // 🔴 Very unsafe
```

## Findings

### Why This Is Dangerous

1. **No Validation**: Assumes data structure without verification
2. **Bypasses Type System**: `as unknown as` completely defeats TypeScript
3. **Runtime Errors Possible**: Malformed messages cause crashes
4. **No Schema Enforcement**: Backend changes break frontend silently

### Impact
- **Severity**: IMPORTANT - Type safety is critical for TypeScript
- **Confidence**: 90/100 - Clear type safety violation

### Evidence
- Location: `hooks/use-watcher-websocket.ts:132-146`
- Uses `unknown` intermediate type (still unsafe)
- No runtime validation or Zod schema

## Proposed Solutions

### Option 1: Define Strict Interface with Zod Validation (Recommended)

```typescript
import { z } from 'zod';

// Define strict message types
const WSJobMessageSchema = z.object({
  type: z.literal("job"),
  data: z.object({
    id: z.string(),
    title: z.string(),
    reward: z.number(),
    url: z.string(),
    source: z.enum(["rss", "websocket"]),
  }),
  timestamp: z.string(),
});

const WSEventMessageSchema = z.object({
  type: z.enum(["job", "event", "error"]),
  event: z.string().optional(),
  message: z.string().optional(),
});

// Type inference from schema
type WSJobMessage = z.infer<typeof WSJobMessageSchema>;
type WSEventMessage = z.infer<typeof WSEventMessageSchema>;

// In hook:
try {
  const parsed = WSJobMessageSchema.parse(message);
  addJob(parsed.data);
} catch (err) {
  console.error("Invalid job message:", err);
}
```

**Pros**:
- Full type safety with runtime validation
- Catches malformed messages gracefully
- Clear error logging
- Schema documents expected structure

**Cons**:
- Requires Zod dependency
- More verbose code
- Need to define all message types

**Effort**: Medium
**Risk**: Low

### Option 2: Define Strict Interfaces (No Runtime Validation)

```typescript
interface WSJobDataMessage {
  type: "job";
  data: {
    id: string;
    title: string;
    reward: number;
    url: string;
    source: "rss" | "websocket";
    timestamp?: string;
  };
  timestamp: string;
}

interface WSEventMessage {
  type: "event";
  event: string;
  timestamp: string;
}

interface WSErrorMessage {
  type: "error";
  message: string;
  timestamp: string;
}

type WSMessage = WSJobDataMessage | WSEventMessage | WSErrorMessage;

// Type guard
function isJobMessage(msg: WSMessage): msg is WSJobDataMessage {
  return msg.type === "job";
}

// In hook:
const parsed = message as WSMessage;
if (isJobMessage(parsed)) {
  addJob(parsed.data);
}
```

**Pros**:
- Compile-time type safety
- No additional dependencies
- Clear type definitions

**Cons**:
- No runtime validation
- Malformed messages still cause issues
- Type guards can be missed

**Effort**: Small
**Risk**: Medium (no runtime safety)

### Option 3: Discriminated Union with Type Narrowing

```typescript
type WSMessage =
  | { type: "job"; data: JobData; timestamp: string }
  | { type: "event"; event: string; timestamp: string }
  | { type: "error"; message: string; timestamp: string };

// In hook:
switch (message.type) {
  case "job":
    addJob(message.data);
    break;
  case "event":
    console.log("Event:", message.event);
    break;
  case "error":
    console.error("Error:", message.message);
    break;
}
```

**Pros**:
- Exhaustive type checking
- Compiler validates all cases
- Clean pattern

**Cons**:
- Requires refactoring message format
- Need to ensure backend sends discriminated union

**Effort**: Medium
**Risk**: Low

## Recommended Action

**Implement Option 1** - Use Zod for runtime validation with TypeScript type inference.

## Technical Details

### Affected Files
- `frontend/hooks/use-watcher-websocket.ts:47-48, 132-146`
- May need: `frontend/lib/ws-types.ts` (new file for types)

### Dependencies
- `zod` package (if not already installed)

### Acceptance Criteria

- [ ] Strict TypeScript interfaces defined for WebSocket messages
- [ ] Zod schema validation implemented
- [ ] Unsafe `as unknown as` casts removed
- [ ] Runtime errors caught and logged
- [ ] Malformed messages don't crash the app
- [ ] Type inference from Zod schemas
- [ ] Tests for valid and invalid messages

## Work Log

### 2025-12-29
- **Finding**: TypeScript review identified unsafe type casts
- **Analysis**: Confirmed type safety violation in WebSocket handling
- **Decision**: Selected Zod validation approach
- **Status**: Pending implementation

## Resources

- [Zod Documentation](https://zod.dev/)
- [TypeScript Type Guards](https://www.typescriptlang.org/docs/handbook/2/narrowing.html#using-type-predicates)
