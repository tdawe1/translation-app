# cmd/translation-worker/plugins/registry.py
"""
Plugin registry for discovering and managing translation worker plugins.

Supports:
- Auto-discovery from multiple directories
- Dependency validation
- Plugin lifecycle (initialize/shutdown)
- Query by type (parsers, quality checks, etc.)
"""

import importlib.util
import inspect
import sys
from pathlib import Path
from typing import Dict, List, Optional, Type, Any, Callable
from dataclasses import dataclass, field

from .base import (
    Plugin,
    ParserPlugin,
    QualityCheckPlugin,
    PipelineStagePlugin,
    UploadDestinationPlugin,
)


@dataclass
class PluginInfo:
    """Information about a loaded plugin."""
    cls: Type
    instance: Any = None
    module_path: str = ""
    initialized: bool = False
    dependencies_met: bool = True
    missing_dependencies: List[str] = field(default_factory=list)

    @property
    def name(self) -> str:
        """Get plugin name from class attribute."""
        return getattr(self.cls, "name", "unknown")

    @property
    def version(self) -> str:
        """Get plugin version from class attribute."""
        return getattr(self.cls, "version", "0.0.0")


class PluginRegistry:
    """
    Registry for managing translation worker plugins.

    Discovers plugins from configured directories and provides
    methods to query and instantiate them by type.
    """

    def __init__(self, plugin_directories: Optional[List[str]] = None):
        """Initialize plugin registry.

        Args:
            plugin_directories: List of directories to scan for plugins
        """
        self.plugin_directories = plugin_directories or []
        self._plugins: Dict[str, PluginInfo] = {}
        self._parsers: Dict[str, PluginInfo] = {}
        self._quality_checks: Dict[str, PluginInfo] = {}
        self._pipeline_stages: Dict[str, PluginInfo] = {}
        self._uploads: Dict[str, PluginInfo] = {}

    def discover(self) -> Dict[str, PluginInfo]:
        """Discover all plugins from configured directories.

        Returns:
            Dict of plugin name -> PluginInfo
        """
        discovered = {}

        for directory in self.plugin_directories:
            dir_path = Path(directory)
            if not dir_path.exists():
                continue

            # Find all Python files
            for py_file in dir_path.rglob("*.py"):
                if py_file.name.startswith("_"):
                    continue

                # Try to load as module
                plugin_infos = self._load_from_file(py_file)
                for info in plugin_infos:
                    name = info.name
                    if name not in discovered:
                        discovered[name] = info

        # Register discovered plugins
        for name, info in discovered.items():
            self._register_plugin(info)

        return discovered

    def _load_from_file(self, file_path: Path) -> List[PluginInfo]:
        """Load plugins from a Python file.

        Args:
            file_path: Path to Python file

        Returns:
            List of PluginInfo for plugins found in file
        """
        plugins = []

        try:
            # Create module name from file path
            module_name = f"plugin_{file_path.stem}_{id(file_path)}"
            spec = importlib.util.spec_from_file_location(module_name, file_path)

            if spec is None or spec.loader is None:
                return plugins

            module = importlib.util.module_from_spec(spec)
            sys.modules[module_name] = module

            spec.loader.exec_module(module)

            # Find plugin classes
            for name, obj in inspect.getmembers(module, inspect.isclass):
                if self._is_plugin_class(obj):
                    info = self._create_plugin_info(obj, str(file_path))
                    plugins.append(info)

        except Exception as e:
            # Log but don't fail - one bad plugin shouldn't break discovery
            pass

        return plugins

    def _is_plugin_class(self, obj: Any) -> bool:
        """Check if a class is a valid plugin.

        A valid plugin:
        - Is not a Protocol class
        - Has required plugin attributes (name, version, dependencies)
        - Implements at least one plugin protocol
        """
        # Skip Protocol classes
        if getattr(obj, "_is_protocol", False):
            return False

        # Check for required attributes
        if not hasattr(obj, "name") or not hasattr(obj, "version"):
            return False

        if not hasattr(obj, "dependencies"):
            return False

        # Check if implements a plugin protocol
        has_plugin_protocol = (
            self._implements_protocol(obj, ParserPlugin) or
            self._implements_protocol(obj, QualityCheckPlugin) or
            self._implements_protocol(obj, PipelineStagePlugin) or
            self._implements_protocol(obj, UploadDestinationPlugin)
        )

        return has_plugin_protocol

    def _implements_protocol(self, obj: Any, protocol: type) -> bool:
        """Check if object implements the given protocol.

        Args:
            obj: Object to check
            protocol: Protocol class to check against

        Returns:
            True if obj implements the protocol

        Note:
            Due to Python 3.14 limitations with @runtime_checkable Protocols,
            we manually check for required methods and attributes.
        """
        # First try isinstance check (works for some cases)
        try:
            if isinstance(obj, protocol):
                return True
        except TypeError:
            pass

        # Map protocols to their required methods (excluding common base attributes)
        protocol_requirements = {
            ParserPlugin: ["supported_extensions", "parse", "render"],
            QualityCheckPlugin: ["check"],
            PipelineStagePlugin: ["execute"],
            UploadDestinationPlugin: ["upload", "delete"],
        }

        required_methods = protocol_requirements.get(protocol, [])

        # Check if object has all required methods
        for method in required_methods:
            if not hasattr(obj, method):
                return False

        return True

    def _create_plugin_info(self, cls: Type, module_path: str) -> PluginInfo:
        """Create PluginInfo from plugin class.

        Args:
            cls: Plugin class
            module_path: Path to module file

        Returns:
            PluginInfo with dependency check results
        """
        dependencies = getattr(cls, "dependencies", [])
        missing = []

        # Check if dependencies are available
        # find_spec returns None if module not found (doesn't raise exception)
        for dep in dependencies:
            spec = importlib.util.find_spec(dep)
            if spec is None:
                missing.append(dep)

        return PluginInfo(
            cls=cls,
            module_path=module_path,
            dependencies_met=len(missing) == 0,
            missing_dependencies=missing
        )

    def _register_plugin(self, info: PluginInfo) -> None:
        """Register a plugin in the appropriate category.

        Args:
            info: PluginInfo to register
        """
        name = info.name
        self._plugins[name] = info

        # Register in specific categories
        cls = info.cls
        if self._implements_protocol(cls, ParserPlugin):
            self._parsers[name] = info
        if self._implements_protocol(cls, QualityCheckPlugin):
            self._quality_checks[name] = info
        if self._implements_protocol(cls, PipelineStagePlugin):
            self._pipeline_stages[name] = info
        if self._implements_protocol(cls, UploadDestinationPlugin):
            self._uploads[name] = info

    def get_plugin(self, name: str) -> Optional[PluginInfo]:
        """Get plugin info by name.

        Args:
            name: Plugin name

        Returns:
            PluginInfo or None if not found
        """
        return self._plugins.get(name)

    def get_parser(self, name: str) -> Optional[PluginInfo]:
        """Get parser plugin by name.

        Args:
            name: Parser plugin name

        Returns:
            PluginInfo or None if not found
        """
        return self._parsers.get(name)

    def get_parser_for_extension(self, ext: str) -> Optional[str]:
        """Get parser plugin name that handles a file extension.

        Args:
            ext: File extension (e.g., ".pdf", ".pptx")

        Returns:
            Plugin name or None if no parser found
        """
        for name, info in self._parsers.items():
            if not info.initialized:
                continue
            instance = info.instance
            if instance and ext in instance.supported_extensions():
                return name
        return None

    def list_parsers(self) -> Dict[str, PluginInfo]:
        """List all parser plugins.

        Returns:
            Dict of parser name -> PluginInfo
        """
        return self._parsers.copy()

    def list_quality_checks(self) -> Dict[str, PluginInfo]:
        """List all quality check plugins.

        Returns:
            Dict of plugin name -> PluginInfo
        """
        return self._quality_checks.copy()

    def list_pipeline_stages(self) -> Dict[str, PluginInfo]:
        """List all pipeline stage plugins.

        Returns:
            Dict of plugin name -> PluginInfo
        """
        return self._pipeline_stages.copy()

    def list_upload_destinations(self) -> Dict[str, PluginInfo]:
        """List all upload destination plugins.

        Returns:
            Dict of plugin name -> PluginInfo
        """
        return self._uploads.copy()

    def initialize_plugin(self, name: str, config: Dict[str, Any]) -> bool:
        """Initialize a plugin instance.

        Args:
            name: Plugin name
            config: Configuration to pass to plugin

        Returns:
            True if initialization succeeded
        """
        info = self._plugins.get(name)
        if not info:
            return False

        if not info.dependencies_met:
            return False

        try:
            instance = info.cls()
            if hasattr(instance, "initialize"):
                instance.initialize(config)

            info.instance = instance
            info.initialized = True
            return True
        except Exception:
            return False

    def shutdown_plugin(self, name: str) -> bool:
        """Shutdown a plugin instance.

        Args:
            name: Plugin name

        Returns:
            True if shutdown succeeded
        """
        info = self._plugins.get(name)
        if not info or not info.initialized:
            return False

        try:
            instance = info.instance
            if hasattr(instance, "shutdown"):
                instance.shutdown()

            info.instance = None
            info.initialized = False
            return True
        except Exception:
            return False

    def shutdown_all(self) -> None:
        """Shutdown all initialized plugins."""
        for name in list(self._plugins.keys()):
            self.shutdown_plugin(name)
