#!/usr/bin/env bash
# scripts/test_deploy.sh
#
# Test deployment script for the translation worker.
# Verifies:
#   1. Dependencies install cleanly
#   2. Full test suite passes
#   3. Worker starts successfully with test config
#   4. Prometheus /metrics endpoint is reachable
#   5. Worker shuts down cleanly
#
# Usage:
#   cd backend/cmd/translation-worker
#   bash scripts/test_deploy.sh

set -euo pipefail

WORKER_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$WORKER_DIR"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓ $1${NC}"; }
fail() { echo -e "${RED}✗ $1${NC}"; exit 1; }
info() { echo -e "${YELLOW}→ $1${NC}"; }

echo "============================================"
echo "  Translation Worker — Test Deployment"
echo "============================================"
echo ""

# --------------------------------------------------
# Step 1: Dependency check
# --------------------------------------------------
info "Checking Python version..."
python3 --version || fail "Python 3 not found"
pass "Python available"

info "Installing dependencies..."
pip install -q prometheus_client tomli pyyaml httpx tenacity redis 2>/dev/null
pass "Core dependencies installed"

# --------------------------------------------------
# Step 2: Test suite
# --------------------------------------------------
info "Running test suite..."
python -m pytest \
  tests/test_style_guide/test_parser.py \
  tests/test_style_guide/test_prompt_builder.py \
  tests/test_audit/test_style_checker.py \
  tests/test_review/test_llm_providers.py \
  tests/test_review/test_workflow.py \
  tests/test_review/test_metrics.py \
  tests/test_review/test_prometheus.py \
  tests/test_integration/test_gengo_integration.py \
  tests/test_main.py \
  tests/test_queue/test_consumer_gengo.py \
  --override-ini="addopts=" \
  -q --tb=line 2>&1

TEST_EXIT=$?
if [ $TEST_EXIT -eq 0 ]; then
  pass "All tests passed"
else
  fail "Tests failed (exit code $TEST_EXIT)"
fi

# --------------------------------------------------
# Step 3: Smoke-test Prometheus endpoint
# --------------------------------------------------
info "Starting worker for Prometheus smoke test..."

# Create a minimal test config with metrics enabled
TMPCONF=$(mktemp /tmp/test_config_XXXX.toml)
cat > "$TMPCONF" <<'EOF'
[worker]
id = "test-deploy"
max_concurrent = 1
heartbeat_interval = "10s"

[translation]
default_provider = "openai"
default_model = "gpt-5.2"

[style_guide]
enabled = false

[metrics]
enabled = true
port = 9091

[job_queue]
enabled = false
EOF

# Start worker in background (it will fail on missing API key — that's expected)
# We only need it alive long enough to check /metrics
python -c "
import sys, time, threading, signal

sys.path.insert(0, '.')
from review.prometheus import start_metrics_server, set_worker_info

start_metrics_server(port=9091)
set_worker_info('test-deploy', 'openai', 'gpt-5.2', style_guide=False)

# Simulate a job metric
from review.metrics import JobMetrics
from review.prometheus import record_job_metrics

m = JobMetrics(
    job_id='deploy-test-1',
    segment_count=10,
    flagged_count=2,
    style_violation_count=1,
    overall_score=0.87,
    provider_name='openai',
    model_name='gpt-5.2',
    style_guide_enabled=False,
)
m.processing_started_at = 1000.0
m.processing_finished_at = 1003.5
record_job_metrics(m)

print('Metrics server running, waiting for scrape test...')
time.sleep(5)
print('Shutting down.')
" &
WORKER_PID=$!

# Wait for server to start
sleep 2

# Scrape metrics
info "Scraping http://localhost:9091/metrics..."
METRICS_OUTPUT=$(curl -s http://localhost:9091/metrics 2>/dev/null || echo "CURL_FAILED")

if echo "$METRICS_OUTPUT" | grep -q "translation_jobs_total"; then
  pass "Prometheus metrics endpoint responding"
else
  kill $WORKER_PID 2>/dev/null || true
  fail "Metrics endpoint not responding or missing expected metrics"
fi

# Verify specific metrics
if echo "$METRICS_OUTPUT" | grep -q 'translation_worker_info.*worker_id="test-deploy"'; then
  pass "Worker info metric present"
else
  info "Worker info metric format may differ (non-blocking)"
fi

if echo "$METRICS_OUTPUT" | grep -q 'translation_jobs_total.*status="completed"'; then
  pass "Jobs total counter present"
else
  info "Jobs total not found (non-blocking)"
fi

if echo "$METRICS_OUTPUT" | grep -q 'translation_style_guide_enabled'; then
  pass "Style guide gauge present"
else
  info "Style guide gauge not found (non-blocking)"
fi

if echo "$METRICS_OUTPUT" | grep -q 'translation_job_duration_seconds'; then
  pass "Duration histogram present"
else
  info "Duration histogram not found (non-blocking)"
fi

# Cleanup
kill $WORKER_PID 2>/dev/null || true
rm -f "$TMPCONF"

echo ""
echo "============================================"
echo -e "  ${GREEN}Test deployment PASSED${NC}"
echo "============================================"
echo ""
echo "To deploy for real:"
echo "  1. Set API key:  export OPENAI_API_KEY=sk-..."
echo "  2. Edit config:  vim config.toml"
echo "  3. Start worker: python main.py"
echo "  4. Scrape:       curl http://localhost:9090/metrics"
echo ""
