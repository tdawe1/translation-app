# review/prometheus.py
"""
Prometheus metrics for the translation worker.

Exposes counters, histograms, and gauges that Prometheus can scrape
for dashboards and alerting.

Metrics are updated after each job by calling `record_job_metrics()`.
The HTTP server is started separately by `start_metrics_server()`.
"""

import logging
from typing import Optional

from prometheus_client import (
    Counter,
    Gauge,
    Histogram,
    Info,
    start_http_server,
)

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Metric definitions — module-level singletons
# ---------------------------------------------------------------------------

JOBS_TOTAL = Counter(
    "translation_jobs_total",
    "Total translation jobs processed",
    ["status", "provider"],
)

SEGMENTS_TOTAL = Counter(
    "translation_segments_total",
    "Total segments processed",
    ["provider"],
)

SEGMENTS_FLAGGED_TOTAL = Counter(
    "translation_segments_flagged_total",
    "Total segments flagged for review",
    ["provider"],
)

STYLE_VIOLATIONS_TOTAL = Counter(
    "translation_style_violations_total",
    "Total style violations detected",
    ["category"],
)

JOB_DURATION_SECONDS = Histogram(
    "translation_job_duration_seconds",
    "Job processing time in seconds",
    ["provider"],
    buckets=(0.5, 1, 2, 5, 10, 30, 60, 120, 300),
)

JOB_QUALITY_SCORE = Histogram(
    "translation_job_quality_score",
    "Overall quality score per job",
    buckets=(0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 1.0),
)

LATEST_FLAG_RATE = Gauge(
    "translation_latest_flag_rate",
    "Flag rate of the most recent job",
)

LATEST_STYLE_VIOLATION_RATE = Gauge(
    "translation_latest_style_violation_rate",
    "Style violation rate of the most recent job",
)

ACTIVE_JOBS = Gauge(
    "translation_active_jobs",
    "Number of currently active jobs",
)

STYLE_GUIDE_ENABLED = Gauge(
    "translation_style_guide_enabled",
    "Whether the Gengo style guide is enabled (1=yes, 0=no)",
)

WORKER_INFO = Info(
    "translation_worker",
    "Worker instance metadata",
)


def set_worker_info(worker_id: str, provider: str, model: str, style_guide: bool):
    """Set static worker info at startup.

    Args:
        worker_id: Unique worker identifier
        provider: Default translation provider name
        model: Default translation model name
        style_guide: Whether style guide is enabled
    """
    WORKER_INFO.info({
        "worker_id": worker_id,
        "provider": provider,
        "model": model,
    })
    STYLE_GUIDE_ENABLED.set(1 if style_guide else 0)


def record_job_metrics(job_metrics) -> None:
    """Update Prometheus metrics from a completed JobMetrics instance.

    Args:
        job_metrics: A review.metrics.JobMetrics instance
    """
    provider = job_metrics.provider_name or "unknown"

    # Job counter
    JOBS_TOTAL.labels(status="completed", provider=provider).inc()

    # Segment counters
    SEGMENTS_TOTAL.labels(provider=provider).inc(job_metrics.segment_count)
    SEGMENTS_FLAGGED_TOTAL.labels(provider=provider).inc(job_metrics.flagged_count)

    # Style violation counters (per category)
    for category, count in job_metrics.style_violations_by_category.items():
        STYLE_VIOLATIONS_TOTAL.labels(category=category).inc(count)

    # Duration histogram
    if job_metrics.processing_duration_ms is not None:
        duration_s = job_metrics.processing_duration_ms / 1000.0
        JOB_DURATION_SECONDS.labels(provider=provider).observe(duration_s)

    # Quality score histogram
    JOB_QUALITY_SCORE.observe(job_metrics.overall_score)

    # Gauges (latest job)
    LATEST_FLAG_RATE.set(job_metrics.flag_rate)
    LATEST_STYLE_VIOLATION_RATE.set(job_metrics.style_violation_rate)


def record_job_failed(provider: str = "unknown") -> None:
    """Record a failed job in Prometheus.

    Args:
        provider: Provider name for the failed job
    """
    JOBS_TOTAL.labels(status="failed", provider=provider).inc()


def start_metrics_server(port: int = 9090) -> bool:
    """Start the Prometheus metrics HTTP server in a daemon thread.

    Args:
        port: Port to listen on (default: 9090)

    Returns:
        True if server started, False if port unavailable
    """
    try:
        start_http_server(port)
        logger.info(f"Prometheus metrics server started on :{port}")
        print(f"  Metrics: http://0.0.0.0:{port}/metrics")
        return True
    except OSError as e:
        logger.warning(f"Could not start metrics server on :{port}: {e}")
        print(f"  Metrics: failed to bind port {port} ({e})")
        return False
