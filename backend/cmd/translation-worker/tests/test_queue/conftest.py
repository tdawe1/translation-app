# tests/test_queue/conftest.py
"""
Pytest configuration for queue tests.

Sets up sys.path to avoid conflict with built-in 'queue' module.
"""

import sys
from pathlib import Path

# Add worker directory to path BEFORE pytest imports anything
# Tests are in: backend/cmd/translation-worker/tests/test_queue/
# Worker dir is: backend/cmd/translation-worker/
worker_dir = Path(__file__).parent.parent
if str(worker_dir) not in sys.path:
    sys.path.insert(0, str(worker_dir))
