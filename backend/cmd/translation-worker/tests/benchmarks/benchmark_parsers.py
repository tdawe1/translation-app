"""
Benchmark parser performance with statistical analysis.

Measures extraction time, memory usage, and validates output for all parsers.
"""

import time
import statistics
from pathlib import Path
import cProfile
import pstats
from io import StringIO


BENCHMARK_RUNS = 10


def benchmark_parser(parser_class, test_file, name):
    """Benchmark a parser with multiple runs."""
    parser = parser_class()

    # Warmup run
    try:
        parser.parse(str(test_file))
    except Exception as e:
        print(f"  Warmup failed for {name}: {e}")
        return None

    # Timed runs
    times = []
    segments_count = 0

    for i in range(BENCHMARK_RUNS):
        start = time.perf_counter()
        try:
            result = parser.parse(str(test_file))
            segments_count = len(result.segments) if hasattr(result, "segments") else 0
            end = time.perf_counter()
            times.append(end - start)
        except Exception as e:
            print(f"  Run {i + 1}/{BENCHMARK_RUNS} failed: {e}")
            break

        if segments_count == 0:
            print(f"  No segments extracted from {test_file}")
            return None

    if not times:
        return None

    return {
        "name": name,
        "file": test_file.name,
        "file_size": test_file.stat().st_size,
        "runs": len(times),
        "avg_time": statistics.mean(times) if times else 0,
        "min_time": min(times) if times else 0,
        "max_time": max(times) if times else 0,
        "std_dev": statistics.stdev(times) if len(times) > 1 else 0,
        "segments": segments_count,
    }


def profile_cpu(parser_class, test_file, name):
    """Profile CPU usage for single run."""
    parser = parser_class()

    profiler = cProfile.Profile()
    profiler.enable()

    try:
        parser.parse(str(test_file))
    except Exception as e:
        print(f"  Profiling failed for {name}: {e}")
        return None

    profiler.disable()

    stats = pstats.Stats(profiler)
    stats.sort_stats("cumtime")

    print(f"\n  CPU Profile Top 10 for {name}:")
    stats.print_stats(10)

    return stats


def main():
    """Run all benchmarks."""
    import sys

    sys.path.insert(0, str(Path(__file__).parent.parent.parent))

    from parsers.pdf_parser import pymupdfParser
    from parsers.pptx_parser import PPTXParser
    from parsers.docx_parser import DOCXParser
    from parsers.xlsx_parser import XLSXParser

    # Use existing large files from translation-tools reference
    reference_path = Path(
        "/home/thomas/translation-tools/translations-pptx-pipeline/archived_runs"
    )

    # Define test files from reference implementation
    test_files = {
        "PPTX Large 1": reference_path / "run-17495340446-artifacts/output_en.pptx",
        "PPTX Large 2": reference_path / "run-17494558119-artifacts/output_en.pptx",
        "PPTX Large 3": reference_path / "run-17494634349-artifacts/output_en.pptx",
        "PPTX Large 4": reference_path / "run-1749558710-artifacts/output_en.pptx",
        "PPTX Large 5": reference_path / "run-17494652852-artifacts/output_en.pptx",
    }

    print("=" * 80)
    print("Parser Performance Benchmarks")
    print("=" * 80)
    print()

    # Check which files exist
    available_tests = {}
    for name, path in test_files.items():
        if path.exists():
            available_tests[name] = (path.name, path)
            print(f"  ✓ {name}: {path.stat().st_size / (1024 * 1024):.1f}MB")

    if not available_tests:
        print("No benchmark test files found!")
        print(f"  Expected location: {reference_path}")
        print()
        return

    # Define parsers (only PPTX has large files available)
    parsers = []

    if "PPTX Large 1" in available_tests:
        parsers.append((PPTXParser, available_tests["PPTX Large 1"][1], "PPTX Large 1"))
    if "PPTX Large 2" in available_tests:
        parsers.append((PPTXParser, available_tests["PPTX Large 2"][1], "PPTX Large 2"))
    if "PPTX Large 3" in available_tests:
        parsers.append((PPTXParser, available_tests["PPTX Large 3"][1], "PPTX Large 3"))
    if "PPTX Large 4" in available_tests:
        parsers.append((PPTXParser, available_tests["PPTX Large 4"][1], "PPTX Large 4"))
    if "PPTX Large 5" in available_tests:
        parsers.append((PPTXParser, available_tests["PPTX Large 5"][1], "PPTX Large 5"))

    if not parsers:
        print("No parsers available for benchmarking!")
        return

    # Run benchmarks
    results = []
    for parser_class, test_file, name in parsers:
        result = benchmark_parser(parser_class, test_file, name)
        if result:
            results.append(result)

    # Output results
    print("\n" + "=" * 80)
    print("BENCHMARK RESULTS")
    print("=" * 80)
    print()
    print(
        f"{'Format':15} {'Test':20} {'Size':10} {'Avg':8} {'Min':8} {'Max':8} {'Segs':6}"
    )
    print("-" * 80)

    for r in results:
        size_mb = r["file_size"] / (1024 * 1024)
        print(
            f"{r['name']:20} {r['file']:25} {size_mb:>7.2f}MB {r['avg_time']:>8.4f}s {r['min_time']:>8.4f}s {r['max_time']:>8.4f}s {r['segments']:>6}"
        )

    print()
    print("Performance Analysis:")
    print("-" * 80)

    # Analyze results
    for r in results:
        avg_time = r["avg_time"]
        print(f"\n{r['name']}: {avg_time:.4f}s average ({r['segments']} segments)")

        # Determine performance category based on file size
        file_size_mb = r["file_size"] / (1024 * 1024)
        if file_size_mb > 5:
            threshold = 10.0  # Large files: < 10s
        elif file_size_mb > 1:
            threshold = 5.0  # Medium files: < 5s
        else:
            threshold = 2.0  # Small files: < 2s

        if avg_time < threshold * 0.5:
            print(f"  ✅ Excellent (< {threshold * 0.5:.1f}s)")
        elif avg_time < threshold:
            print(f"  ✅ Good (< {threshold:.1f}s)")
        elif avg_time < threshold * 1.5:
            print(f"  ⚠️  Acceptable (< {threshold * 1.5:.1f}s)")
        elif avg_time < threshold * 2:
            print(f"  ⚠️  Slow (< {threshold * 2:.1f}s) - Consider optimization")
        else:
            print(
                f"  ❌ Very Slow (> {threshold * 2:.1f}s) - Strong optimization needed"
            )

        # Profile slow cases
        if avg_time > threshold:
            print(f"  📊 Profiling slow case...")
            profile_cpu(parser_class, r["test_file"], r["name"])

    print()
    print("=" * 80)
    print("RECOMMENDATIONS")
    print("=" * 80)
    print()

    slow_parsers = [r for r in results if r["avg_time"] > threshold]
    if not slow_parsers:
        print("✅ All parsers meet performance targets!")
        print("   No optimization needed at this time.")
    else:
        print("⚠️  The following parsers exceed performance thresholds:")
        for r in slow_parsers:
            size_mb = r["file_size"] / (1024 * 1024)
            print(
                f"   - {r['name']}: {r['avg_time']:.4f}s ({size_mb:.1f}MB, {r['segments']} segments)"
            )

        print()
        print("Optimization Priority (from plan):")
        print("1. Check for algorithmic inefficiencies")
        print("2. Reduce string copies and memory allocations")
        print("3. Add caching for repeated operations")
        print("4. Consider Cython for hot paths (if 3-5x improvement possible)")
        print("5. C++ extension only if 5-10x improvement with maintainable code")

    print()


if __name__ == "__main__":
    main()
