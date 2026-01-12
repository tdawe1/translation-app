#!/usr/bin/env python3
"""
Benchmark PPTX reconstruction (parsing + layout adjustment + rebuilding).

This measures the ACTUAL bottleneck: parsing → layout adjustment → rebuilding.
"""

import os
import sys
import time
import zipfile
import xml.etree.ElementTree as ET
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from layout.preserver import LayoutPreserver, Rectangle, Font


def extract_text_from_paragraph(
    p_el, ns="{http://schemas.openxmlformats.org/drawingml/2006/main"
):
    """Extract text from a paragraph element."""
    t_tag = ns + "t"
    r_tag = ns + "r"

    parts = []
    for node in p_el:
        if node.tag == r_tag:
            t = node.find(t_tag)
            parts.append("" if t is None or t.text is None else t.text)

    return "".join(parts)


def replace_paragraph_text(
    p_el, new_text, ns="{http://schemas.openxmlformats.org/drawingml/2006/main}"
):
    """Replace paragraph text while preserving run structure."""
    t_tag = ns + "t"
    r_tag = ns + "r"

    runs = [child for child in p_el if child.tag == r_tag]
    if not runs:
        r = ET.Element(r_tag)
        t = ET.SubElement(r, t_tag)
        t.text = ""
        p_el.insert(0, r)
        runs = [r]

    N = len(runs)
    L = len(new_text)
    if N == 1:
        chunks = [new_text]
    else:
        base = L // N
        rem = L % N
        chunks = []
        start = 0
        for i in range(N):
            size = base + (1 if i < rem else 0)
            chunks.append(new_text[start : start + size])
            start += size

    for r, chunk in zip(runs, chunks):
        t = r.find(t_tag)
        if t is None:
            t = ET.SubElement(r, t_tag)
        t.text = chunk

    # Clear extra runs
    for r in runs[len(chunks) :]:
        t = r.find(t_tag)
        if t is not None:
            t.text = ""


def simulate_reconstruction(
    zip_path: str, preserver: LayoutPreserver, apply_layout: bool = True
) -> dict:
    """Simulate full reconstruction workflow."""
    results = {
        "total_slides": 0,
        "total_paragraphs": 0,
        "parse_time": 0,
        "extract_time": 0,
        "layout_time": 0,
        "replace_time": 0,
        "serialize_time": 0,
        "total_time": 0,
    }

    start_total = time.time()

    A_NS = "{http://schemas.openxmlformats.org/drawingml/2006/main}"

    with zipfile.ZipFile(zip_path, "r") as zin:
        slide_files = [
            n
            for n in zin.namelist()
            if n.startswith("ppt/slides/slide") and n.endswith(".xml")
        ]

        results["total_slides"] = len(slide_files)

        for sf in slide_files:
            # Parse
            parse_start = time.time()
            data = zin.read(sf)
            root = ET.fromstring(data)
            parse_time = time.time() - parse_start
            results["parse_time"] += parse_time

            # Extract text
            extract_start = time.time()
            paragraphs = list(root.iter(A_NS + "p"))
            texts = []
            for p in paragraphs:
                text = extract_text_from_paragraph(p)
                if text.strip():
                    texts.append((p, text))
            extract_time = time.time() - extract_start
            results["extract_time"] += extract_time

            results["total_paragraphs"] += len(texts)

            if apply_layout:
                # Simulate layout adjustment
                layout_start = time.time()
                for p, text in texts:
                    # Assume JA->EN translation (text shrinks)
                    translated_text = f"[Translated] {text}"  # Placeholder

                    # Calculate font adjustment (simplified)
                    font_size = 12.0
                    bounds = Rectangle(width=200.0, height=50.0)
                    new_size = preserver.suggest_font_size(
                        source_text=text,
                        target_text=translated_text,
                        container_width=bounds.width,
                        current_font_size=font_size,
                    )
                layout_time = time.time() - layout_start
                results["layout_time"] += layout_time

                # Replace text
                replace_start = time.time()
                for p, text in texts:
                    # Simulate translation with text reduction
                    translated_text = f"[Translated] {text}"
                    replace_paragraph_text(p, translated_text)

                    # Simulate font size change
                    font_size = 12.0
                    bounds = Rectangle(width=200.0, height=50.0)
                    new_size = preserver.suggest_font_size(
                        source_text=text,
                        target_text=translated_text,
                        container_width=bounds.width,
                        current_font_size=font_size,
                    )
                replace_time = time.time() - replace_start
                results["replace_time"] += replace_time

            # Serialize (rebuild XML)
            serialize_start = time.time()
            _ = ET.tostring(root, encoding="utf-8", xml_declaration=True)
            serialize_time = time.time() - serialize_start
            results["serialize_time"] += serialize_time

    results["total_time"] = time.time() - start_total

    return results


def format_results(results: dict, file_name: str) -> str:
    """Format benchmark results for display."""
    file_size = os.path.getsize(file_name) / (1024 * 1024)  # MB

    output = []
    output.append(f"\n{'=' * 70}")
    output.append(f"File: {os.path.basename(file_name)}")
    output.append(f"Size: {file_size:.2f} MB")
    output.append(f"Slides: {results['total_slides']}")
    output.append(f"Paragraphs: {results['total_paragraphs']}")
    output.append(f"{'=' * 70}")
    output.append("\nTiming Breakdown:")
    output.append(
        f"  Parse XML:          {results['parse_time']:.3f}s ({results['parse_time'] / results['total_time'] * 100:.1f}%)"
    )
    output.append(
        f"  Extract Text:       {results['extract_time']:.3f}s ({results['extract_time'] / results['total_time'] * 100:.1f}%)"
    )
    output.append(
        f"  Layout Adjustment:  {results['layout_time']:.3f}s ({results['layout_time'] / results['total_time'] * 100:.1f}%)"
    )
    output.append(
        f"  Replace Text:       {results['replace_time']:.3f}s ({results['replace_time'] / results['total_time'] * 100:.1f}%)"
    )
    output.append(
        f"  Serialize XML:      {results['serialize_time']:.3f}s ({results['serialize_time'] / results['total_time'] * 100:.1f}%)"
    )
    output.append("-" * 70)
    output.append(f"  TOTAL:             {results['total_time']:.3f}s")
    output.append(f"\nPer-Slide Averages:")
    if results["total_slides"] > 0:
        avg_per_slide = results["total_time"] / results["total_slides"]
        output.append(f"  Average:           {avg_per_slide:.3f}s per slide")
        output.append(
            f"  Est. for 50 slides: {avg_per_slide * 50:.1f}s ({avg_per_slide * 50 / 60:.1f} min)"
        )

    return "\n".join(output)


def find_test_pptx_files(reference_dir: str) -> list:
    """Find PPTX files in reference directory."""
    pptx_dir = Path(reference_dir)
    if not pptx_dir.exists():
        return []

    pptx_files = []
    # Look for output_en.pptx files in archived_runs
    for pptx_file in pptx_dir.rglob("output_en.pptx"):
        pptx_files.append(str(pptx_file))

    return sorted(pptx_files)


def main():
    """Run reconstruction benchmarks."""
    # Reference directory with test files
    ref_dir = "/home/thomas/translation-tools/translations-pptx-pipeline/archived_runs"

    # Find test files
    test_files = find_test_pptx_files(ref_dir)

    if not test_files:
        print(f"No PPTX files found in {ref_dir}")
        return

    print(f"Found {len(test_files)} PPTX files for benchmarking")
    print(f"Using first 3 files for reconstruction benchmark\n")

    # Create layout preserver
    preserver = LayoutPreserver(strategy="autofit")

    # Benchmark with and without layout adjustment
    all_results = []
    for test_file in test_files[:3]:
        print(f"\nProcessing: {test_file}")

        # Without layout adjustment (baseline)
        print("  Benchmarking without layout adjustment...")
        results_baseline = simulate_reconstruction(
            test_file, preserver, apply_layout=False
        )
        print(format_results(results_baseline, test_file))

        # With layout adjustment
        print("  Benchmarking with layout adjustment...")
        results_with_layout = simulate_reconstruction(
            test_file, preserver, apply_layout=True
        )
        print(format_results(results_with_layout, test_file))

        # Calculate layout overhead
        layout_overhead = (
            results_with_layout["total_time"] - results_baseline["total_time"]
        )
        print(f"\nLayout Adjustment Overhead: {layout_overhead:.3f}s")

        all_results.append(
            {
                "file": test_file,
                "baseline": results_baseline,
                "with_layout": results_with_layout,
                "layout_overhead": layout_overhead,
            }
        )

    # Summary
    print(f"\n{'=' * 70}")
    print("SUMMARY")
    print(f"{'=' * 70}")

    for i, res in enumerate(all_results, 1):
        file_size = os.path.getsize(res["file"]) / (1024 * 1024)
        print(f"\n{i}. {os.path.basename(res['file'])}")
        print(f"   Size: {file_size:.2f} MB")
        print(f"   Baseline (no layout): {res['baseline']['total_time']:.3f}s")
        print(f"   With layout:          {res['with_layout']['total_time']:.3f}s")
        print(
            f"   Layout overhead:      {res['layout_overhead']:.3f}s ({res['layout_overhead'] / res['baseline']['total_time'] * 100:.1f}%)"
        )
        print(f"   Slides:              {res['baseline']['total_slides']}")
        print(
            f"   Avg per slide:        {res['baseline']['total_time'] / res['baseline']['total_slides']:.3f}s"
        )

        # Estimate for 50 slides
        avg_per_slide = res["baseline"]["total_time"] / res["baseline"]["total_slides"]
        est_50 = avg_per_slide * 50
        print(f"   Est. for 50 slides:   {est_50:.1f}s ({est_50 / 60:.1f} min)")


if __name__ == "__main__":
    main()
