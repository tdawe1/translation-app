# tests/test_plugins/test_registry.py
"""
Unit tests for plugin registry.

Tests plugin discovery, loading, and management.
"""

import pytest
import sys
import tempfile
from pathlib import Path

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from plugins.base import (
    Plugin,
    ParserPlugin,
    Segment,
    ParsedDocument,
)
from plugins.registry import PluginRegistry


class TestPluginRegistry:
    """Test PluginRegistry functionality."""

    def test_empty_registry(self):
        """Registry should start empty."""
        registry = PluginRegistry([])
        assert len(registry._plugins) == 0
        assert len(registry._parsers) == 0

    def test_register_parser_manually(self):
        """Should manually register a parser plugin."""
        # Create a test parser class
        class TestParser:
            name = "test_parser"
            version = "1.0"
            dependencies = []

            def supported_extensions(self):
                return [".test"]

            def parse(self, file_path):
                return ParsedDocument(
                    segments=[Segment(id="1", text="test", context={})],
                    metadata={},
                    format="test"
                )

            def render(self, doc, output_path):
                pass

        registry = PluginRegistry([])

        # Create plugin info manually and register
        from plugins.registry import PluginInfo
        info = PluginInfo(cls=TestParser, module_path="test.py")
        registry._register_plugin(info)

        assert "test_parser" in registry._parsers
        assert registry.get_parser("test_parser") is not None

    def test_initialize_plugin(self):
        """Should initialize plugin with config."""
        class ConfigurableParser:
            name = "configurable"
            version = "1.0"
            dependencies = []
            initialized_config = None

            def supported_extensions(self):
                return [".cfg"]

            def parse(self, file_path):
                return ParsedDocument(
                    segments=[],
                    metadata={},
                    format="cfg"
                )

            def render(self, doc, output_path):
                pass

            def initialize(self, config):
                self.initialized_config = config

        registry = PluginRegistry([])
        from plugins.registry import PluginInfo
        info = PluginInfo(cls=ConfigurableParser, module_path="test.py")
        registry._register_plugin(info)

        config = {"setting": "value"}
        success = registry.initialize_plugin("configurable", config)

        assert success is True
        assert info.initialized is True
        assert info.instance.initialized_config == config

    def test_shutdown_plugin(self):
        """Should shutdown plugin instance."""
        class TestPlugin:
            name = "test"
            version = "1.0"
            dependencies = []
            shutdown_called = False

            def initialize(self, config):
                pass

            def shutdown(self):
                self.shutdown_called = True

        registry = PluginRegistry([])
        from plugins.registry import PluginInfo
        info = PluginInfo(cls=TestPlugin, module_path="test.py")

        # Create and set instance
        instance = TestPlugin()
        info.instance = instance
        info.initialized = True
        registry._plugins["test"] = info

        # Shutdown
        success = registry.shutdown_plugin("test")

        assert success is True
        assert instance.shutdown_called is True
        assert info.initialized is False
        assert info.instance is None

    def test_get_parser_for_extension(self):
        """Should find parser by file extension."""
        class PDFParser:
            name = "pdf_parser"
            version = "1.0"
            dependencies = []

            def supported_extensions(self):
                return [".pdf"]

            def parse(self, file_path):
                return ParsedDocument(
                    segments=[],
                    metadata={},
                    format="pdf"
                )

            def render(self, doc, output_path):
                pass

        class PPTXParser:
            name = "pptx_parser"
            version = "1.0"
            dependencies = []

            def supported_extensions(self):
                return [".pptx", ".ppt"]

            def parse(self, file_path):
                return ParsedDocument(
                    segments=[],
                    metadata={},
                    format="pptx"
                )

            def render(self, doc, output_path):
                pass

        registry = PluginRegistry([])

        # Register parsers
        from plugins.registry import PluginInfo
        pdf_info = PluginInfo(cls=PDFParser, module_path="test.py")
        pptx_info = PluginInfo(cls=PPTXParser, module_path="test.py")
        registry._register_plugin(pdf_info)
        registry._register_plugin(pptx_info)

        # Initialize them
        registry.initialize_plugin("pdf_parser", {})
        registry.initialize_plugin("pptx_parser", {})

        # Find by extension
        assert registry.get_parser_for_extension(".pdf") == "pdf_parser"
        assert registry.get_parser_for_extension(".pptx") == "pptx_parser"
        assert registry.get_parser_for_extension(".ppt") == "pptx_parser"
        assert registry.get_parser_for_extension(".docx") is None

    def test_discover_from_directory(self):
        """Should discover plugins from a directory."""
        # Create temporary directory with a test plugin
        with tempfile.TemporaryDirectory() as tmpdir:
            tmpdir = Path(tmpdir)

            # Write a test plugin file
            plugin_code = '''
class TestDiscoveryParser:
    name = "discovered_parser"
    version = "1.0"
    dependencies = []

    def supported_extensions(self):
        return [".disco"]

    def parse(self, file_path):
        from plugins.base import ParsedDocument, Segment
        return ParsedDocument(
            segments=[Segment(id="1", text="discovered", context={})],
            metadata={},
            format="disco"
        )

    def render(self, doc, output_path):
        pass
'''
            plugin_file = tmpdir / "test_plugin.py"
            plugin_file.write_text(plugin_code)

            # Discover plugins
            registry = PluginRegistry([str(tmpdir)])
            discovered = registry.discover()

            assert "discovered_parser" in discovered
            assert "discovered_parser" in registry._parsers


class TestDependencyValidation:
    """Test dependency validation in plugins."""

    def test_missing_dependencies_marked(self):
        """Should mark plugins with missing dependencies."""
        class PluginWithDeps:
            name = "needs_module"
            version = "1.0"
            dependencies = ["totally_fake_module_xyz", "another_fake_one"]

            def supported_extensions(self):
                return [".test"]

            def parse(self, file_path):
                return ParsedDocument(
                    segments=[],
                    metadata={},
                    format="test"
                )

            def render(self, doc, output_path):
                pass

        registry = PluginRegistry([])
        from plugins.registry import PluginInfo
        info = registry._create_plugin_info(PluginWithDeps, "test.py")

        assert info.dependencies_met is False
        assert "totally_fake_module_xyz" in info.missing_dependencies
        assert "another_fake_one" in info.missing_dependencies

    def test_no_dependencies_always_met(self):
        """Plugins with no dependencies should always pass validation."""
        class SimplePlugin:
            name = "simple"
            version = "1.0"
            dependencies = []

            def supported_extensions(self):
                return [".simple"]

            def parse(self, file_path):
                return ParsedDocument(
                    segments=[],
                    metadata={},
                    format="simple"
                )

            def render(self, doc, output_path):
                pass

        registry = PluginRegistry([])
        from plugins.registry import PluginInfo
        info = registry._create_plugin_info(SimplePlugin, "test.py")

        assert info.dependencies_met is True
        assert len(info.missing_dependencies) == 0


class TestPluginInfo:
    """Test PluginInfo dataclass."""

    def test_plugin_info_properties(self):
        """Should extract name and version from class."""
        class MyPlugin:
            name = "my_plugin"
            version = "2.5.0"
            dependencies = []

        from plugins.registry import PluginInfo
        info = PluginInfo(cls=MyPlugin, module_path="test.py")

        assert info.name == "my_plugin"
        assert info.version == "2.5.0"
