# XLSX Parser Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement an XLSX parser plugin for the translation worker to extract translatable text from spreadsheets and render translated versions while preserving layout and formatting.

**Architecture:** The parser will implement the `ParserPlugin` protocol using `openpyxl`. It will iterate through worksheets and cells, extracting string values into `Segment` objects with coordinate context (e.g., "Sheet1!A1") and style metadata, then use the original file as a template for rendering translations.

**Tech Stack:** Python 3.12+, `openpyxl`

---

### Task 1: Environment Setup

**Files:**
- Modify: `backend/cmd/translation-worker/requirements.txt`

**Step 1: Add dependency to requirements.txt**

```text
openpyxl>=3.1.2
```

**Step 2: Run installation**

Run: `pip install -r backend/cmd/translation-worker/requirements.txt`
Expected: Successfully installs `openpyxl`

**Step 3: Commit**

```bash
git add backend/cmd/translation-worker/requirements.txt
git commit -m "chore: add openpyxl dependency for xlsx parser"
```

---

### Task 2: Basic XLSXParser Structure

**Files:**
- Create: `backend/cmd/translation-worker/parsers/xlsx_parser.py`
- Create: `backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py`

**Step 1: Write the failing test for plugin attributes**

```python
import pytest
from parsers.xlsx_parser import XLSXParser

def test_xlsx_parser_attributes():
    parser = XLSXParser()
    assert parser.name == "xlsx_parser"
    assert parser.version == "1.0.0"
    assert "openpyxl" in parser.dependencies
```

**Step 2: Run test to verify it fails**

Run: `pytest backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py`
Expected: FAIL with `ImportError: cannot import name 'XLSXParser'`

**Step 3: Write minimal implementation**

```python
# parsers/xlsx_parser.py
from typing import List, Dict, Any, Optional
import logging

logger = logging.getLogger(__name__)

try:
    import openpyxl
    OPENPYXL_AVAILABLE = True
except ImportError:
    OPENPYXL_AVAILABLE = False

class XLSXParser:
    name = "xlsx_parser"
    version = "1.0.0"
    dependencies = ["openpyxl"]

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        if not OPENPYXL_AVAILABLE:
            raise RuntimeError("openpyxl is not installed")
        self.config = config or {}

    def initialize(self, config: Dict[str, Any]) -> None:
        self.config.update(config)

    def shutdown(self) -> None:
        pass

    def supported_extensions(self) -> List[str]:
        return [".xlsx", ".xlsm", ".xltx", ".xltm"]
```

**Step 4: Run test to verify it passes**

Run: `pytest backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/translation-worker/parsers/xlsx_parser.py backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py
git commit -m "feat: add basic XLSXParser structure"
```

---

### Task 3: Simple Parsing (Single Sheet)

**Files:**
- Modify: `backend/cmd/translation-worker/parsers/xlsx_parser.py`
- Modify: `backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py`

**Step 1: Write the failing test for parsing**

```python
import tempfile
from pathlib import Path
from openpyxl import Workbook
from plugins import Segment

def test_parse_simple_xlsx():
    parser = XLSXParser()
    with tempfile.NamedTemporaryFile(suffix=".xlsx", delete=False) as tmp:
        wb = Workbook()
        ws = wb.active
        ws["A1"] = "Hello World"
        wb.save(tmp.name)
        
        try:
            parsed = parser.parse(tmp.name)
            assert len(parsed.segments) == 1
            assert parsed.segments[0].text == "Hello World"
            assert parsed.segments[0].context["coordinate"] == "A1"
        finally:
            Path(tmp.name).unlink()
```

**Step 2: Run test to verify it fails**

Run: `pytest backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py`
Expected: FAIL with `AttributeError: 'XLSXParser' object has no attribute 'parse'`

**Step 3: Write minimal implementation for parse**

```python
# In parsers/xlsx_parser.py
from openpyxl import load_workbook
from plugins import ParsedDocument, Segment

    def parse(self, file_path: str) -> ParsedDocument:
        wb = load_workbook(file_path, data_only=True)
        segments = []
        
        for sheet_name in wb.sheetnames:
            ws = wb[sheet_name]
            for row in ws.iter_rows():
                for cell in row:
                    if cell.value and isinstance(cell.value, str):
                        segments.append(Segment(
                            id=f"{sheet_name}!{cell.coordinate}",
                            text=cell.value,
                            context={
                                "type": "cell",
                                "sheet_name": sheet_name,
                                "coordinate": cell.coordinate,
                                "row": cell.row,
                                "column": cell.column
                            }
                        ))
        
        return ParsedDocument(
            segments=segments,
            metadata={"sheet_count": len(wb.sheetnames)},
            format="xlsx",
            source_path=file_path
        )
```

**Step 4: Run test to verify it passes**

Run: `pytest backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/translation-worker/parsers/xlsx_parser.py
git commit -m "feat: implement basic XLSX parsing"
```

---

### Task 4: Formatting and Multi-sheet Parsing

**Files:**
- Modify: `backend/cmd/translation-worker/parsers/xlsx_parser.py`
- Modify: `backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py`

**Step 1: Write the failing test for formatting and multi-sheet**

```python
def test_parse_complex_xlsx():
    parser = XLSXParser()
    with tempfile.NamedTemporaryFile(suffix=".xlsx", delete=False) as tmp:
        wb = Workbook()
        ws1 = wb.active
        ws1.title = "Sheet1"
        ws1["A1"] = "Bold Text"
        ws1["A1"].font = openpyxl.styles.Font(bold=True)
        
        ws2 = wb.create_sheet("Sheet2")
        ws2["B2"] = "Italic Text"
        ws2["B2"].font = openpyxl.styles.Font(italic=True)
        wb.save(tmp.name)
        
        try:
            parsed = parser.parse(tmp.name)
            assert len(parsed.segments) == 2
            
            s1 = next(s for s in parsed.segments if s.context["sheet_name"] == "Sheet1")
            assert s1.metadata["bold"] is True
            
            s2 = next(s for s in parsed.segments if s.context["sheet_name"] == "Sheet2")
            assert s2.metadata["italic"] is True
        finally:
            Path(tmp.name).unlink()
```

**Step 2: Run test to verify it fails**

Run: `pytest backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py`
Expected: FAIL on `assert s1.metadata["bold"] is True` (KeyError or False)

**Step 3: Update implementation to extract styles**

```python
# In XLSXParser.parse, update the loop:
                    if cell.value and isinstance(cell.value, str):
                        metadata = {
                            "bold": cell.font.bold if cell.font else False,
                            "italic": cell.font.italic if cell.font else False,
                            "font_name": cell.font.name if cell.font else None,
                            "font_size": cell.font.sz if cell.font else None,
                        }
                        segments.append(Segment(
                            id=f"{sheet_name}!{cell.coordinate}",
                            text=cell.value,
                            context={...}, # Same as before
                            metadata=metadata
                        ))
```

**Step 4: Run test to verify it passes**

Run: `pytest backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/translation-worker/parsers/xlsx_parser.py
git commit -m "feat: extract formatting metadata from XLSX cells"
```

---

### Task 5: Rendering Support

**Files:**
- Modify: `backend/cmd/translation-worker/parsers/xlsx_parser.py`
- Modify: `backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py`

**Step 1: Write the failing test for rendering**

```python
def test_render_xlsx():
    parser = XLSXParser()
    with tempfile.NamedTemporaryFile(suffix=".xlsx", delete=False) as src_tmp:
        with tempfile.NamedTemporaryFile(suffix=".xlsx", delete=False) as out_tmp:
            wb = Workbook()
            ws = wb.active
            ws["A1"] = "Original"
            wb.save(src_tmp.name)
            
            try:
                parsed = parser.parse(src_tmp.name)
                parsed.segments[0].text = "Translated"
                
                parser.render(parsed, out_tmp.name, template_path=src_tmp.name)
                
                # Verify output
                out_wb = load_workbook(out_tmp.name)
                assert out_wb.active["A1"].value == "Translated"
            finally:
                Path(src_tmp.name).unlink()
                Path(out_tmp.name).unlink()
```

**Step 2: Run test to verify it fails**

Run: `pytest backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py`
Expected: FAIL with `AttributeError: 'XLSXParser' object has no attribute 'render'`

**Step 3: Implement render method**

```python
# In parsers/xlsx_parser.py

    def render(self, doc: ParsedDocument, output_path: str, template_path: Optional[str] = None) -> None:
        if not template_path:
            # For simplicity in this worker, we usually require a template to preserve styles
            wb = openpyxl.Workbook()
            ws = wb.active
            for segment in doc.segments:
                sheet_name = segment.context.get("sheet_name", "Sheet")
                if sheet_name not in wb.sheetnames:
                    wb.create_sheet(sheet_name)
                ws = wb[sheet_name]
                ws[segment.context["coordinate"]] = segment.text
        else:
            wb = load_workbook(template_path)
            for segment in doc.segments:
                sheet_name = segment.context.get("sheet_name")
                if sheet_name in wb.sheetnames:
                    ws = wb[sheet_name]
                    ws[segment.context["coordinate"]] = segment.text
        
        wb.save(output_path)
```

**Step 4: Run test to verify it passes**

Run: `pytest backend/cmd/translation-worker/tests/test_parsers/test_xlsx_parser.py`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/translation-worker/parsers/xlsx_parser.py
git commit -m "feat: implement XLSX rendering"
```

---

### Task 6: Cleanup and Registration

**Files:**
- Modify: `backend/cmd/translation-worker/parsers/__init__.py`

**Step 1: Export XLSXParser in __init__.py**

```python
from .xlsx_parser import XLSXParser, create_xlsx_parser
```

**Step 2: Add factory function to xlsx_parser.py**

```python
def create_xlsx_parser(config: Optional[Dict[str, Any]] = None) -> XLSXParser:
    return XLSXParser(config)
```

**Step 3: Commit**

```bash
git add backend/cmd/translation-worker/parsers/
git commit -m "feat: register XLSXParser in parsers module"
```
