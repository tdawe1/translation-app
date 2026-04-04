# Code Review: Gengo Style Guide Integration + Prometheus Metrics

**Date**: Jan 2026
**Reviewer**: E1
**Scope**: All files modified or created across Phases 1–5 + Prometheus

---

## Files Reviewed

### New files
| File | Lines | Purpose |
|------|-------|---------|
| `review/metrics.py` | 105 | Job-level metrics dataclass |
| `review/prometheus.py` | 167 | Prometheus metric definitions + HTTP server |
| `tests/test_review/test_metrics.py` | 96 | Unit tests for JobMetrics |
| `tests/test_review/test_prometheus.py` | 168 | Unit + integration tests for Prometheus |
| `tests/test_queue/test_consumer_gengo.py` | 173 | Queue consumer integration tests |
| `scripts/test_deploy.sh` | 130 | Deployment verification script |

### Modified files
| File | Nature of change |
|------|-----------------|
| `main.py` | Runtime helpers, Prometheus startup, active_jobs gauge |
| `review/workflow.py` | Style checker + metrics + Prometheus in process_job |
| `review/models.py` | `style_issues` field on TranslationSegment |
| `review/exporter.py` | `style_issues` column in CSV, json import |
| `review/flagging.py` | (unchanged) |
| `config.toml` | Added `[metrics]` section |
| `config.example.toml` | Added `[metrics]` section, fixed path |
| `requirements.txt` | Added `prometheus_client` |
| `docs/GENGO_STYLE_GUIDE.md` | Full rewrite |
| `tests/test_main.py` | 19 new tests for runtime helpers |
| `tests/test_review/test_workflow.py` | 6 style checker tests, fixed 5 pre-existing failures |
| `tests/test_integration/test_gengo_integration.py` | 3 new workflow integration tests |

---

## Review Findings

### Resolved during review

1. **Unused imports in `prometheus.py`**: `Thread`, `CollectorRegistry`, `REGISTRY` were imported but unused. Removed.
2. **Return type mismatch**: `start_metrics_server` was annotated `Optional[Thread]` but returned `True`/`None`. Fixed to `bool`.
3. **ACTIVE_JOBS gauge unused**: Was declared but never incremented/decremented. Wired into `_process_job` enter/exit.

### Accepted design decisions

4. **Prometheus import in try/except**: `workflow.py` and `main.py` import Prometheus inside `try/except` blocks. This is intentional — Prometheus is optional infrastructure and should never crash a translation job.

5. **Module-level Prometheus singletons**: Standard pattern for `prometheus_client`. Metrics register themselves on import. Tests share the global registry, but test assertions use delta checks (`after - before`) to avoid cross-test pollution.

6. **`flagged_count` overwrite**: In `process_job`, `metrics.record_flag()` increments `flagged_count` inside the loop, then `metrics.flagged_count = job.flagged_count` overwrites after the loop. Both values should be identical. The overwrite is defensive normalization against the authoritative `job.update_metrics()` count. The `record_flag()` call is still needed because it populates `flag_reasons` list.

7. **Style checker flag logic**: When a segment is already flagged by the confidence flagger, the style checker still records `style_issues` but doesn't overwrite `flag_reason`. This is correct — the segment is already in review, and `style_issues` are available as supplementary audit data.

8. **`getattr` in exporter**: `_segment_to_row` uses `getattr(segment, 'style_issues', [])` to handle segment objects from older code paths that may lack the field. Defensive but appropriate for a multi-path system.

### Potential future concerns (non-blocking)

9. **Prometheus in multi-process workers**: If the worker is scaled with `multiprocessing`, Prometheus counters won't aggregate correctly across processes. This requires `prometheus_client.multiprocess` mode. Not relevant for the current single-process architecture.

10. **Histogram bucket tuning**: Duration buckets (0.5s–300s) and quality score buckets (0.1–1.0) are reasonable defaults. May need tuning after observing real production distributions.

11. **`LATEST_FLAG_RATE` gauge race**: If multiple jobs complete near-simultaneously (concurrent consumer), the gauge reflects whichever job finished last. This is expected behavior for a "latest" gauge.

---

## Test Coverage

| Area | Tests | Status |
|------|-------|--------|
| Style guide parser | 5 | Pass |
| Prompt builder | 2 | Pass |
| Style checker | 26 | Pass |
| LLM providers | 11 | Pass |
| Workflow (core) | 30 | Pass |
| Workflow (style checker) | 6 | Pass |
| Metrics | 12 | Pass |
| Prometheus | 10 | Pass |
| Integration (Gengo) | 9 | Pass |
| Main (runtime helpers) | 42 | Pass |
| Queue consumer | 7 | Pass |
| **Total** | **197** | **All pass** |

### What's NOT covered (and why)

- **Real API calls**: All provider calls use mocks. This is correct for unit/integration tests. Real provider testing requires live API keys.
- **Redis integration**: Queue consumer tests mock `job_manager`. Real Redis tests require a running Redis instance.
- **Concurrent job processing**: Not tested. Would require threading/async test infrastructure.
- **`main()` end-to-end**: Tested up to workflow construction. The full event loop requires Redis and is tested via `scripts/test_deploy.sh`.

---

## Verdict

**Approved.** The changeset is clean, well-tested, and follows the smallest-correct-change principle. Prometheus integration is properly optional and won't affect translation reliability.
