# Bilingual Translation Review Workflow - Design Document

**Date:** 2025-01-12
**Status:** Design Complete, Implementation Pending
**MVP Scope:** All components stubbed, core functionality implemented

---

## Overview

A translation review system where **automated multi-model translation** produces a final output that a **human approves via web UI**. The bilingual CSV serves as an audit trail and fine-tuning data source, not as the primary review interface.

### Key Principles

1. **Human time is valuable** - Only surface segments that genuinely need attention
2. **Automated decisions first** - Judge model resolves most disagreements without human input
3. **Final approval gate** - Human sees completed result and says "ship it" or "fix this"
4. **Configurable by project** - Critical documents block; routine documents flow async

---

## Architecture

```
Source File → Multi-Model Translation → Judge Model → Flagging Engine → Output Generation
                                                                                │
                                                                                ▼
                                                                        ┌───────────────┐
                                                                        │  Web UI:      │
                                                                        │  Approve/     │
                                                                        │  Reject       │
                                                                        └───────┬───────┘
                                                                                │
                                                                                ▼
                                                                        ┌───────────────┐
                                                                        │  Export/      │
                                                                        │  CSV Audit    │
                                                                        └───────────────┘
```

---

## Components

### 1. Multi-Model Translation

**Purpose:** Generate competing translations for judge evaluation

```python
@dataclass
class TranslationCandidate:
    model_name: str
    text: str
    confidence: float
    glossary_matches: List[str]
    latency_ms: int
```

**Models configured:**
- Model A: Primary (e.g., claude-4.5-sonnet)
- Model B: Secondary (e.g., gpt-4o)
- Optional: Model C for additional coverage

### 2. Judge Model

**Purpose:** Evaluate competing translations and select the best option

**Input:**
- Source text
- Translation candidates (A, B)
- Context (document type, segment position)
- Glossary terms used

**Output (JSON):**
```json
{
  "winner": "A" | "B" | "tie",
  "confidence": 0.0-1.0,
  "reasoning": "explanation",
  "concerns": ["issue1", "issue2"],
  "suggested_edits": "merged version if tie"
}
```

**Stub behavior:** In MVP, returns random selection with placeholder reasoning. Full implementation uses sophisticated prompting.

### 3. Flagging Engine

**Purpose:** Identify segments needing human attention

**Signals:**
- Judge confidence < threshold (default: 0.7)
- Judge returned "tie" with concerns
- Length ratio anomaly (target/source < 0.5 or > 2.0)
- Manual flag from user
- Random sample (configurable rate)

**Review priority score:**
```python
review_priority = (
    (1.0 - judge_confidence) * 0.5 +
    anomaly_score * 0.3 +
    (1.0 if manual_flag else 0.0) * 0.2
)
```

**Thresholds:**
- `> 0.7`: Block output (pre-commit review required)
- `0.3 - 0.7`: Include in review (async approval)
- `< 0.3`: Auto-approve

### 4. Web UI

**Pages:**

**Dashboard** - List of translation jobs
```tsx
// columns: status, progress, score, flagged count, submitted time
{jobs.map(job => (
  <TranslationJobRow
    job={job}
    onApprove={() => handleApprove(job.id)}
    onReview={() => navigate(`/review/${job.id}`)}
  />
))}
```

**Review Page** - Detailed view for approval
- Quality summary (score, segments, judge resolutions)
- Flagged segments (collapsed, expandable)
- In-place editing for flagged segments
- Preview pane (rendered output)
- Approve/Reject buttons

**Stub behavior:** Static UI with mock data, no real backend integration in MVP.

### 5. Data Models

```python
@dataclass
class TranslationJob:
    id: str
    source_file: str
    target_file: str
    status: Literal["processing", "pending_approval", "approved", "rejected", "exported"]
    overall_score: float
    segment_count: int
    flagged_count: int
    judge_resolutions: int
    created_at: datetime
    completed_at: Optional[datetime]
    approved_at: Optional[datetime]
    approved_by: Optional[str]

@dataclass
class TranslationSegment:
    id: str
    job_id: str
    source: str
    target: str
    context: Dict
    judge_winner: Literal["model_a", "model_b", "edited", "tie"]
    judge_confidence: float
    judge_reasoning: str
    is_flagged: bool
    flag_reason: Optional[str]
    model_a_output: Optional[str]
    model_b_output: Optional[str]
    glossary_terms: List[str]

@dataclass
class JudgeResult:
    segment_id: str
    winner: str
    confidence: float
    reasoning: str
    concerns: List[str]
    suggested_edits: Optional[str]
```

### 6. Configuration

```toml
[judge]
enabled = true
model = "claude-4.5-sonnet"
timeout_sec = 30
fallback_on_timeout = true

[flagging]
auto_approve_threshold = 0.80
block_threshold = 0.70
random_sample_rate = 0.02

[output.bilingual_csv]
enabled = true
path = "/watch/bilingual/"
encoding = "utf-8-sig"
include_judge_reasoning = true
include_alternatives = true

[project_types.critical]
approval_mode = "blocking"
auto_approve_threshold = 0.95
random_sample_rate = 0.10

[project_types.routine]
approval_mode = "async"
auto_approve_threshold = 0.85
random_sample_rate = 0.02
```

---

## CSV Format (Audit Trail)

```csv
segment_id,source,target,judge_winner,judge_confidence,judge_reasoning,is_flagged,flag_reason,model_a,model_b,glossary_terms,context
s1,こんにちは,Hello,model_a,0.95,Both accurate and natural,0,,Hello,Hi,,"{""slide"":1}"
s2,苦情処理,Complaint handling,tie,0.65,Technical nuance disagreement,1,Glossary conflict,Complaint handling,Customer support,,"{""slide"":2}"
```

---

## API Endpoints (Backend)

```go
// Job management
GET    /api/v1/translation/jobs          // List jobs
GET    /api/v1/translation/jobs/:id      // Get job details
POST   /api/v1/translation/jobs/:id/approve  // Approve job
POST   /api/v1/translation/jobs/:id/reject   // Reject job
POST   /api/v1/translation/jobs/:id/segments/:id/edit  // Edit segment

// Review data
GET    /api/v1/translation/jobs/:id/segments  // Get all segments
GET    /api/v1/translation/jobs/:id/flagged  // Get only flagged segments
GET    /api/v1/translation/jobs/:id/preview  // Get preview URL
```

---

## MVP Implementation Status

| Component | MVP Scope | Full Implementation |
|-----------|-----------|---------------------|
| Multi-model translation | ✅ Stub with 2 hard-coded models | Configurable model registry |
| Judge model | ✅ Stub returns random winner | Sophisticated prompting, editing |
| Flagging engine | ✅ Confidence-based only | All signals implemented |
| Web UI | ✅ Static with mock data | Real backend integration |
| CSV export | ✅ Basic format | All fields included |
| Project config | ✅ Single profile | Per-project profiles |

---

## Implementation Order (MVP)

1. **Backend stubs** - TranslationJob, TranslationSegment models
2. **API endpoints** - Jobs list, details, approve/reject (stubbed logic)
3. **Web UI skeleton** - Dashboard, review page with mock data
4. **Judge stub** - Random selection with logging
5. **Flagging stub** - Simple confidence-based flagging
6. **CSV export** - Basic bilingual CSV generation
7. **Integration** - Wire stubbed components together

---

## Open Questions

- [ ] Judge model prompt refinement (domain-specific, few-shot)
- [ ] Judge editing authority and validation
- [ ] Preview rendering for each document type (PDF/DOCX/PPTX)
- [ ] Concurrent review handling (multiple users, same job)
- [ ] Learning from human corrections (glossary updates, fine-tuning)
