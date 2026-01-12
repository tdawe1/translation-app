# review/__init__.py
"""
Bilingual translation review workflow system.

Provides multi-model translation with judge-based selection,
flagging for human review, and bilingual CSV export for audit trail.

## Components

- **models**: Core data structures (TranslationJob, TranslationSegment, JudgeResult)
- **multimodel**: Coordinate translation across multiple models
- **judge**: Evaluate competing translations and select winner
- **flagging**: Identify segments needing human review
- **exporter**: Export to bilingual CSV format

## Quick Start

```python
from review import (
    MultiModelTranslator,
    TranslationJudge,
    FlaggingEngine,
    BilingualCSVExporter,
    TranslationJob,
    ReviewConfig
)

# Create components
translator = MultiModelTranslator()
judge = TranslationJudge()
flagger = FlaggingEngine()
exporter = BilingualCSVExporter()

# Process a translation
candidates = translator.translate("Hello, world")
result = judge.judge("seg1", "Hello, world", candidates)

# Create a job with segments
job = TranslationJob(id="job1", source_file="in.txt", target_file="out.txt")
# ... add segments to job ...

# Export to CSV
exporter.export_job(job)
```

## MVP Status

All components are stubbed for MVP:
- MultiModelTranslator: Returns placeholder translations
- TranslationJudge: Random selection with stubbed reasoning
- FlaggingEngine: Confidence-based flagging functional
- BilingualCSVExporter: Full CSV export implementation

## CLI Usage

For command-line interface documentation, see README.md in this directory.
"""

from .models import (
    TranslationCandidate,
    JudgeResult,
    TranslationSegment,
    TranslationJob,
    ReviewConfig,
)

from .multimodel import (
    MultiModelTranslator,
    create_multimodel_translator,
)

from .judge import (
    TranslationJudge,
    create_judge,
)

from .flagging import (
    FlaggingEngine,
    create_flagging_engine,
)

from .exporter import (
    BilingualCSVExporter,
    create_exporter,
)

from .workflow import (
    TranslationWorkflow,
    ReviewWorkflowBuilder,
    create_workflow,
)

__all__ = [
    # Models
    "TranslationCandidate",
    "JudgeResult",
    "TranslationSegment",
    "TranslationJob",
    "ReviewConfig",
    # Multi-model
    "MultiModelTranslator",
    "create_multimodel_translator",
    # Judge
    "TranslationJudge",
    "create_judge",
    # Flagging
    "FlaggingEngine",
    "create_flagging_engine",
    # Exporter
    "BilingualCSVExporter",
    "create_exporter",
    # Workflow
    "TranslationWorkflow",
    "ReviewWorkflowBuilder",
    "create_workflow",
]
