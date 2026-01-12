# cmd/translation-worker/plugins/__init__.py
"""
Plugin system for translation worker.

Uses Protocol-based structural typing for extensibility:
- No explicit inheritance required
- @runtime_checkable enables isinstance() validation
- Plugins can be discovered from multiple directories
"""

from .base import (
    Plugin,
    ParserPlugin,
    QualityCheckPlugin,
    PipelineStagePlugin,
    UploadDestinationPlugin,
    Segment,
    ParsedDocument,
    QualityReport,
    QualityIssue,
    StageResult,
    UploadResult,
    Severity,
    REQUIRED_PLUGIN_ATTRIBUTES,
    has_plugin_attributes,
    is_plugin,
)
from .registry import PluginRegistry

__all__ = [
    "Plugin",
    "ParserPlugin",
    "QualityCheckPlugin",
    "PipelineStagePlugin",
    "UploadDestinationPlugin",
    "Segment",
    "ParsedDocument",
    "QualityReport",
    "QualityIssue",
    "StageResult",
    "UploadResult",
    "Severity",
    "REQUIRED_PLUGIN_ATTRIBUTES",
    "has_plugin_attributes",
    "is_plugin",
    "PluginRegistry",
]
