#!/usr/bin/env python3
"""Triage Analyzer - Post-mortem analysis for catfight triage runs

Provides tools for analyzing completed triage runs without hitting GitHub API.
Works with local JSONL logs and result files.
"""

import json
import os
import re
import sys
from collections import defaultdict
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Tuple

# Import from logger module
from triage_logger import (
    JSONL_DIR, RAW_DIR, list_runs, load_run_summary,
    get_model_benchmarks, ensure_dirs
)


def parse_results_file(filepath: Path) -> Dict:
    """Parse a catfight results file into structured data"""
    with open(filepath) as f:
        content = f.read()

    result = {
        "models": [],
        "winner": None,
        "responses": {}
    }

    # Extract model results
    pattern = r'>>> \[(\d+)/(\d+)\] (\S+) \(([^)]+)\)\s+(?:DONE (\d+\.?\d*)s \| (\d+) tokens \| (\d+\.?\d*) tok/s|(FAILED|ERROR)[^\n]*)'

    for match in re.finditer(pattern, content):
        idx, total, name, model, time, tokens, toks, error = match.groups()

        model_result = {
            "name": name,
            "model": model,
            "status": "done" if time else "failed"
        }

        if time:
            model_result["time"] = float(time)
            model_result["tokens"] = int(tokens)
            model_result["toks"] = float(toks)

        result["models"].append(model_result)

    # Find winner
    winner_match = re.search(r'WINNER: (\S+) wins with ([\d.]+)s', content)
    if winner_match:
        result["winner"] = {"name": winner_match.group(1), "time": float(winner_match.group(2))}

    # Extract responses (between ### ModelName and next ### or end)
    response_pattern = r'### (\S+) \(([^)]+)\)\n-+\n(.*?)(?=\n### |\n$|\Z)'
    for match in re.finditer(response_pattern, content, re.DOTALL):
        name, model, response = match.groups()
        result["responses"][model] = response.strip()

    return result


def extract_scope_from_response(response: str) -> Optional[str]:
    """Extract scope estimate from a model response"""
    patterns = [
        r'\*\*Size:\*\*\s*(XS|S|M|L|XL)',
        r'\*\*Scope[^:]*:\*\*\s*(XS|S|M|L|XL)',
        r'Size:\s*(XS|S|M|L|XL)',
        r'Scope:\s*(XS|S|M|L|XL)',
    ]

    for pattern in patterns:
        match = re.search(pattern, response, re.IGNORECASE)
        if match:
            return match.group(1).upper()
    return None


def get_scope_consensus(results: Dict) -> Tuple[Optional[str], Dict[str, int]]:
    """Determine scope consensus from model responses"""
    scope_votes = defaultdict(int)

    for model, response in results.get("responses", {}).items():
        scope = extract_scope_from_response(response)
        if scope:
            scope_votes[scope] += 1

    if not scope_votes:
        return None, {}

    # Find most common
    sorted_votes = sorted(scope_votes.items(), key=lambda x: x[1], reverse=True)
    top_scope, top_count = sorted_votes[0]
    total_votes = sum(scope_votes.values())

    # Need majority or plurality
    if top_count >= total_votes / 2:
        return top_scope, dict(scope_votes)

    return None, dict(scope_votes)


def analyze_run(run_dir: Path) -> Dict:
    """Analyze a complete triage run from raw results"""
    analysis = {
        "issues": [],
        "model_stats": defaultdict(lambda: {"runs": 0, "time": 0, "tokens": 0, "failures": 0}),
        "scope_distribution": defaultdict(int),
        "errors": []
    }

    for results_file in sorted(run_dir.glob("issue-*-results.txt")):
        try:
            issue_num = int(re.search(r'issue-(\d+)-results', results_file.name).group(1))
            results = parse_results_file(results_file)

            # Get scope consensus
            scope, votes = get_scope_consensus(results)
            if scope:
                analysis["scope_distribution"][scope] += 1

            issue_data = {
                "number": issue_num,
                "winner": results.get("winner"),
                "scope": scope,
                "scope_votes": votes,
                "models_run": len(results["models"]),
                "models_succeeded": sum(1 for m in results["models"] if m["status"] == "done")
            }
            analysis["issues"].append(issue_data)

            # Aggregate model stats
            for m in results["models"]:
                stats = analysis["model_stats"][m["model"]]
                if m["status"] == "done":
                    stats["runs"] += 1
                    stats["time"] += m.get("time", 0)
                    stats["tokens"] += m.get("tokens", 0)
                else:
                    stats["failures"] += 1

        except Exception as e:
            analysis["errors"].append(f"{results_file.name}: {str(e)}")

    # Calculate averages
    for model, stats in analysis["model_stats"].items():
        if stats["runs"] > 0:
            stats["avg_time"] = round(stats["time"] / stats["runs"], 2)
            stats["avg_tokens"] = round(stats["tokens"] / stats["runs"])

    return analysis


def print_analysis_report(analysis: Dict, verbose: bool = False):
    """Print formatted analysis report"""
    issues = analysis["issues"]
    model_stats = analysis["model_stats"]

    print("=" * 70)
    print("TRIAGE RUN ANALYSIS")
    print("=" * 70)
    print()

    # Summary
    print(f"Issues Processed: {len(issues)}")
    succeeded = sum(1 for i in issues if i["models_succeeded"] == i["models_run"])
    print(f"Fully Successful: {succeeded}/{len(issues)}")
    print()

    # Scope distribution
    print("SCOPE DISTRIBUTION:")
    print("-" * 30)
    scope_dist = analysis["scope_distribution"]
    total = sum(scope_dist.values())
    for scope in ["XS", "S", "M", "L", "XL"]:
        count = scope_dist.get(scope, 0)
        bar = "█" * int(count / max(total, 1) * 20)
        print(f"  {scope:>2}: {bar:<20} {count:>3} ({count/max(total,1)*100:.0f}%)")
    print()

    # Model performance
    print("MODEL PERFORMANCE:")
    print("-" * 70)
    print(f"{'Model':<35} {'Runs':>5} {'Avg Time':>10} {'Avg Tok':>8} {'Fail':>5}")
    print("-" * 70)

    sorted_models = sorted(model_stats.items(), key=lambda x: x[1].get("avg_time", 0), reverse=True)
    for model, stats in sorted_models:
        print(f"{model:<35} {stats['runs']:>5} {stats.get('avg_time', 0):>9.1f}s {stats.get('avg_tokens', 0):>8} {stats['failures']:>5}")
    print()

    # Recommendations
    print("RECOMMENDATIONS:")
    print("-" * 30)

    # Find slow models
    slow_models = [m for m, s in model_stats.items() if s.get("avg_time", 0) > 100]
    if slow_models:
        print(f"  ❌ SLOW (>100s): {', '.join(slow_models)}")

    # Find unreliable models
    unreliable = [m for m, s in model_stats.items()
                  if s["failures"] > 0 and s["failures"] / (s["runs"] + s["failures"]) > 0.1]
    if unreliable:
        print(f"  ❌ UNRELIABLE (>10% fail): {', '.join(unreliable)}")

    # Find fast & reliable
    fast_reliable = [m for m, s in model_stats.items()
                     if s.get("avg_time", 999) < 50 and s["failures"] == 0 and s["runs"] > 5]
    if fast_reliable:
        print(f"  ✅ FAST & RELIABLE: {', '.join(fast_reliable)}")

    # Best value (fast + good output)
    best_value = [(m, s) for m, s in model_stats.items()
                  if s.get("avg_time", 999) < 40 and s.get("avg_tokens", 0) > 400 and s["failures"] == 0]
    if best_value:
        bv = sorted(best_value, key=lambda x: x[1]["avg_tokens"], reverse=True)[0]
        print(f"  ⭐ BEST VALUE: {bv[0]} ({bv[1]['avg_time']}s, {bv[1]['avg_tokens']} tokens)")

    print()

    if verbose and analysis["errors"]:
        print("ERRORS:")
        for err in analysis["errors"]:
            print(f"  - {err}")


def export_analysis_json(analysis: Dict, output_path: Path):
    """Export analysis to JSON file"""
    # Convert defaultdicts to regular dicts
    export = {
        "issues": analysis["issues"],
        "model_stats": dict(analysis["model_stats"]),
        "scope_distribution": dict(analysis["scope_distribution"]),
        "errors": analysis["errors"]
    }

    with open(output_path, "w") as f:
        json.dump(export, f, indent=2)

    print(f"Exported to: {output_path}")


def main():
    import argparse

    parser = argparse.ArgumentParser(description="Analyze triage run results")
    parser.add_argument("command", choices=["analyze", "list", "benchmark", "export"],
                       help="Command to run")
    parser.add_argument("--run-dir", "-d", type=Path,
                       help="Path to triage run directory (for analyze/export)")
    parser.add_argument("--output", "-o", type=Path,
                       help="Output file path (for export)")
    parser.add_argument("--verbose", "-v", action="store_true",
                       help="Verbose output")
    parser.add_argument("--json", action="store_true",
                       help="Output as JSON")

    args = parser.parse_args()

    if args.command == "list":
        runs = list_runs()
        if args.json:
            print(json.dumps(runs, indent=2))
        else:
            print(f"Found {len(runs)} triage runs:\n")
            for r in runs[:20]:
                dur = r.get('duration', 0)
                print(f"  {r['run_id']} - {r['issues']} issues, {dur/60:.1f}min")

    elif args.command == "benchmark":
        benchmarks = get_model_benchmarks()
        if args.json:
            print(json.dumps(benchmarks, indent=2))
        else:
            print("Model Benchmarks (aggregated):\n")
            print(f"{'Model':<40} {'Runs':>6} {'Avg Time':>10} {'Fail %':>8}")
            print("-" * 70)
            for model, stats in sorted(benchmarks.items(), key=lambda x: x[1].get("avg_time", 0), reverse=True):
                print(f"{model:<40} {stats['runs']:>6} {stats.get('avg_time', 0):>9.1f}s {stats.get('failure_rate', 0):>7.1f}%")

    elif args.command == "analyze":
        if not args.run_dir:
            # Try to find most recent run in /tmp
            tmp_runs = list(Path("/tmp").glob("catfight-triage-*"))
            if tmp_runs:
                args.run_dir = sorted(tmp_runs)[-1]
                print(f"Using: {args.run_dir}\n")
            else:
                print("Error: No --run-dir specified and no runs found in /tmp")
                sys.exit(1)

        analysis = analyze_run(args.run_dir)

        if args.json:
            print(json.dumps({
                "issues": analysis["issues"],
                "model_stats": dict(analysis["model_stats"]),
                "scope_distribution": dict(analysis["scope_distribution"])
            }, indent=2))
        else:
            print_analysis_report(analysis, args.verbose)

    elif args.command == "export":
        if not args.run_dir or not args.output:
            print("Error: --run-dir and --output required for export")
            sys.exit(1)

        analysis = analyze_run(args.run_dir)
        export_analysis_json(analysis, args.output)


if __name__ == "__main__":
    main()
