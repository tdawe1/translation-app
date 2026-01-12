"""
Generate benchmark test files for parser performance testing.

Creates representative files in small/medium/large sizes with realistic content.
"""

from pathlib import Path
import time


def create_benchmark_pdf(output_dir: Path, size: str = "medium"):
    """Create PDF test file of specified size."""
    size_configs = {
        "small": (1, 5, "pdf_bench_small.pdf"),
        "medium": (10, 20, "pdf_bench_medium.pdf"),
        "large": (50, 100, "pdf_bench_large.pdf"),
    }

    pages_min, pages_max, filename = size_configs[size]
    pages = (pages_min + pages_max) // 2

    print(f"Would create PDF: {filename} ({pages} pages)")
    pdf_path = output_dir / filename

    return pdf_path


def create_benchmark_pptx(output_dir: Path, size: str = "medium"):
    """Create PPTX test file of specified size."""
    size_configs = {
        "small": (5, 10, "pptx_bench_small.pptx"),
        "medium": (20, 30, "pptx_bench_medium.pptx"),
        "large": (50, 75, "pptx_bench_large.pptx"),
    }

    slides_min, slides_max, filename = size_configs[size]
    slides = (slides_min + slides_max) // 2

    print(f"Would create PPTX: {filename} ({slides} slides)")
    pptx_path = output_dir / filename

    return pptx_path


def create_benchmark_docx(output_dir: Path, size: str = "medium"):
    """Create DOCX test file of specified size."""
    size_configs = {
        "small": (1, 2, "docx_bench_small.docx"),
        "medium": (5, 10, "docx_bench_medium.docx"),
        "large": (20, 30, "docx_bench_large.docx"),
    }

    pages_min, pages_max, filename = size_configs[size]
    pages = (pages_min + pages_max) // 2

    print(f"Would create DOCX: {filename} ({pages} pages)")
    docx_path = output_dir / filename

    return docx_path


def create_benchmark_xlsx(output_dir: Path, size: str = "medium"):
    """Create XLSX test file of specified size."""
    size_configs = {
        "small": (50, 100, "xlsx_bench_small.xlsx"),
        "medium": (500, 1000, "xlsx_bench_medium.xlsx"),
        "large": (2000, 3000, "xlsx_bench_large.xlsx"),
    }

    cells_min, cells_max, filename = size_configs[size]
    cells = (cells_min + cells_max) // 2

    print(f"Would create XLSX: {filename} ({cells} cells)")
    xlsx_path = output_dir / filename

    return xlsx_path


def main():
    """Plan to create all benchmark test files."""
    output_dir = Path(__file__).parent / "data"
    output_dir.mkdir(parents=True, exist_ok=True)

    print("Benchmark Test File Generation Plan:")
    print()

    for size in ["small", "medium", "large"]:
        print(f"--- {size.upper()} FILES ---")
        print(f"  PDF: {create_benchmark_pdf(output_dir, size).name}")
        print(f"  PPTX: {create_benchmark_pptx(output_dir, size).name}")
        print(f"  DOCX: {create_benchmark_docx(output_dir, size).name}")
        print(f"  XLSX: {create_benchmark_xlsx(output_dir, size).name}")
        print()

    print("Test file generation plan created successfully!")
    print(f"Location: {output_dir}")
    print()
    print("To create actual files, implement PDF/PPTX/DOCX/XLSX writers")
    print("using appropriate libraries:")
    print("  - PDF: reportlab or fitz (pymupdf)")
    print("  - PPTX: python-pptx")
    print("  - DOCX: python-docx")
    print("  - XLSX: openpyxl")


if __name__ == "__main__":
    main()
