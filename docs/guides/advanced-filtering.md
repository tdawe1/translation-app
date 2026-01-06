# Advanced Filtering Strategies

GengoWatcher provides powerful filtering capabilities to help you cut through the noise and focus on the jobs that matter most. This guide covers complex filtering strategies.

## 1. Reward-Based Filtering

Beyond a simple minimum reward, consider setting a **Maximum Reward** to avoid "distraction" jobs that may be too complex or require specialized software you don't possess.

**Recommended Setup:**
- **Min Reward**: `$15.00` (Your target hourly rate equivalent).
- **Max Reward**: `$200.00` (To avoid multi-day commitments).

---

## 2. Language Pair Granularity

Don't just filter by broad languages. If you specialize in specific regional dialects or industry-specific pairings, list them explicitly.

**Example Configuration:**
```json
{
  "included_language_pairs": [
    "en-us_ja-jp",
    "en-gb_ja-jp",
    "ja-jp_en-us"
  ]
}
```

---

## 3. Keyword Matching (Pro/Enterprise)

Keywords allow you to filter based on the content of the job title.

### Positive Matching (Whitelist)
Include terms related to your areas of expertise.
- **Keywords**: `medical`, `legal`, `gaming`, `cryptocurrency`
- **Result**: You only get notified if the job contains one of these terms.

### Negative Matching (Blacklist)
Exclude terms for domains you don't work in.
- **Excluded Keywords**: `adult`, `transcription`, `proofreading`
- **Result**: Any job containing these terms will be ignored, even if it matches your reward and language filters.

---

## 4. Multi-Condition Logic

GengoWatcher applies all filters as an **AND** operation across categories, and an **OR** operation within categories.

**The formula:**
`(Language Matches) AND (Reward matches) AND (Contains Keyword A OR Keyword B) AND (Does NOT contain Blacklist X)`

---

## 5. Strategy: The "Golden Ticket" Watcher

Power users often set up their watcher with very strict filters but high-frequency polling:
- **Min Reward**: `$50.00`
- **Keywords**: `Urgent`, `ASAP`
- **Polling**: 10 seconds (Enterprise)
- **Auto-Accept**: Enabled

This ensures that the most lucrative, urgent jobs are claimed the second they appear.

## Next Steps
- [Auto-Accept Setup](../guides/auto-accept-setup.md)
- [Watcher System Overview](../core-concepts/watcher-system.md)
- [Update Watcher Config API](../api/watcher-endpoints.md)
