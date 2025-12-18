#!/usr/bin/env python3
"""Triage Logger - Compact JSONL logging for catfight triage runs

Provides streaming, disk-efficient logging with automatic retention management.
"""

import json
import os
import gzip
import shutil
from datetime import datetime, timedelta
from pathlib import Path
from typing import Optional, Dict, Any, List

# Default paths
CLOOD_DIR = Path.home() / ".clood"
TRIAGE_DIR = CLOOD_DIR / "triage"
JSONL_DIR = TRIAGE_DIR / "logs"
RAW_DIR = TRIAGE_DIR / "raw"
ARCHIVE_DIR = TRIAGE_DIR / "archive"

# Retention settings
RAW_RETENTION_DAYS = 7
COMPRESS_AFTER_DAYS = 1


def ensure_dirs():
    """Create triage directories if they don't exist"""
    for d in [CLOOD_DIR, TRIAGE_DIR, JSONL_DIR, RAW_DIR, ARCHIVE_DIR]:
        d.mkdir(parents=True, exist_ok=True)


class TriageLogger:
    """Streaming JSONL logger for triage runs"""

    def __init__(self, run_id: Optional[str] = None):
        ensure_dirs()
        self.run_id = run_id or datetime.now().strftime("%Y%m%d-%H%M%S")
        self.log_path = JSONL_DIR / f"triage-{self.run_id}.jsonl"
        self.summary_path = JSONL_DIR / f"triage-{self.run_id}-summary.json"
        self.start_time = datetime.now()
        self.issues_processed = 0
        self.model_stats: Dict[str, Dict] = {}

    def log_model_result(self, issue_num: int, model: str,
                         time_sec: float, tokens: int, tok_sec: float,
                         status: str = "done", error: Optional[str] = None):
        """Log a single model result (append to JSONL)"""
        entry = {
            "ts": datetime.now().isoformat(),
            "issue": issue_num,
            "model": model,
            "time": round(time_sec, 2),
            "tokens": tokens,
            "toks": round(tok_sec, 2),
            "status": status
        }
        if error:
            entry["error"] = error[:200]  # Truncate long errors

        with open(self.log_path, "a") as f:
            f.write(json.dumps(entry) + "\n")

        # Update running stats
        if model not in self.model_stats:
            self.model_stats[model] = {"runs": 0, "time": 0, "tokens": 0, "failures": 0}

        stats = self.model_stats[model]
        if status == "done":
            stats["runs"] += 1
            stats["time"] += time_sec
            stats["tokens"] += tokens
        else:
            stats["failures"] += 1

    def log_issue_complete(self, issue_num: int, title: str,
                          winner_model: str, scope_consensus: Optional[str] = None):
        """Log issue completion"""
        entry = {
            "ts": datetime.now().isoformat(),
            "event": "issue_complete",
            "issue": issue_num,
            "title": title[:100],
            "winner": winner_model,
            "scope": scope_consensus
        }
        with open(self.log_path, "a") as f:
            f.write(json.dumps(entry) + "\n")
        self.issues_processed += 1

    def finalize(self):
        """Write final summary and return stats"""
        duration = (datetime.now() - self.start_time).total_seconds()

        summary = {
            "run_id": self.run_id,
            "start_time": self.start_time.isoformat(),
            "duration_sec": round(duration, 1),
            "issues_processed": self.issues_processed,
            "model_stats": {}
        }

        for model, stats in self.model_stats.items():
            runs = stats["runs"]
            summary["model_stats"][model] = {
                "runs": runs,
                "failures": stats["failures"],
                "avg_time": round(stats["time"] / runs, 2) if runs else 0,
                "avg_tokens": round(stats["tokens"] / runs) if runs else 0,
                "total_time": round(stats["time"], 1)
            }

        with open(self.summary_path, "w") as f:
            json.dump(summary, f, indent=2)

        return summary


def format_github_summary(issue_num: int, title: str, model_results: List[Dict],
                          winner_response: str, scope_consensus: Optional[str] = None) -> str:
    """Format a compact GitHub comment with triage summary

    Args:
        issue_num: GitHub issue number
        title: Issue title
        model_results: List of {model, time, tokens, toks, status} dicts
        winner_response: Full response from winning model
        scope_consensus: Agreed scope estimate (XS/S/M/L/XL)

    Returns:
        Markdown-formatted comment string
    """
    # Build results table
    table_rows = []
    winner_model = None
    winner_time = float('inf')

    for r in model_results:
        status = "✅" if r.get("status") == "done" else "❌"
        time_str = f"{r['time']:.1f}s" if r.get("time") else "-"
        tokens_str = str(r.get("tokens", "-"))
        toks_str = f"{r['toks']:.1f}" if r.get("toks") else "-"

        table_rows.append(f"| {r['model'][:25]} | {time_str} | {tokens_str} | {toks_str} | {status} |")

        if r.get("status") == "done" and r.get("time", float('inf')) < winner_time:
            winner_time = r["time"]
            winner_model = r["model"]

    scope_line = f"**Scope Consensus:** {scope_consensus}\n" if scope_consensus else ""

    comment = f"""## 🐱 Catfight Triage Summary

{scope_line}**Winner:** {winner_model} ({winner_time:.1f}s)
**Models:** {len(model_results)} | **Issue:** #{issue_num}

| Model | Time | Tokens | Tok/s | Status |
|-------|------|--------|-------|--------|
{chr(10).join(table_rows)}

<details>
<summary>📝 Winning Response ({winner_model})</summary>

{winner_response[:6000]}

</details>

---
*Triage by clood catfight - compact summary mode*
"""
    return comment


def cleanup_old_data():
    """Remove old raw data, compress old logs"""
    ensure_dirs()
    now = datetime.now()
    cleaned = {"deleted": 0, "compressed": 0}

    # Clean raw results older than retention period
    if RAW_DIR.exists():
        for item in RAW_DIR.iterdir():
            if item.is_dir():
                try:
                    # Parse date from directory name
                    dir_date = datetime.strptime(item.name[:8], "%Y%m%d")
                    age = (now - dir_date).days

                    if age > RAW_RETENTION_DAYS:
                        shutil.rmtree(item)
                        cleaned["deleted"] += 1
                except (ValueError, IndexError):
                    pass

    # Compress old JSONL logs
    if JSONL_DIR.exists():
        for item in JSONL_DIR.glob("*.jsonl"):
            try:
                file_date = datetime.strptime(item.name.split("-")[1][:8], "%Y%m%d")
                age = (now - file_date).days

                if age > COMPRESS_AFTER_DAYS and not item.suffix == ".gz":
                    # Compress
                    with open(item, 'rb') as f_in:
                        with gzip.open(str(item) + '.gz', 'wb') as f_out:
                            shutil.copyfileobj(f_in, f_out)
                    item.unlink()
                    cleaned["compressed"] += 1
            except (ValueError, IndexError):
                pass

    return cleaned


def load_run_summary(run_id: str) -> Optional[Dict]:
    """Load summary for a specific run"""
    summary_path = JSONL_DIR / f"triage-{run_id}-summary.json"
    if summary_path.exists():
        with open(summary_path) as f:
            return json.load(f)
    return None


def list_runs() -> List[Dict]:
    """List all triage runs with basic stats"""
    runs = []
    ensure_dirs()

    for item in sorted(JSONL_DIR.glob("*-summary.json"), reverse=True):
        try:
            with open(item) as f:
                summary = json.load(f)
                runs.append({
                    "run_id": summary.get("run_id"),
                    "start_time": summary.get("start_time"),
                    "issues": summary.get("issues_processed", 0),
                    "duration": summary.get("duration_sec", 0)
                })
        except (json.JSONDecodeError, KeyError):
            pass

    return runs


def get_model_benchmarks(run_ids: Optional[List[str]] = None) -> Dict[str, Dict]:
    """Aggregate model performance across runs"""
    ensure_dirs()

    if run_ids is None:
        # Use all runs
        summaries = list(JSONL_DIR.glob("*-summary.json"))
    else:
        summaries = [JSONL_DIR / f"triage-{rid}-summary.json" for rid in run_ids]

    aggregated: Dict[str, Dict] = {}

    for path in summaries:
        if not path.exists():
            continue
        try:
            with open(path) as f:
                summary = json.load(f)

            for model, stats in summary.get("model_stats", {}).items():
                if model not in aggregated:
                    aggregated[model] = {"runs": 0, "failures": 0, "total_time": 0, "total_tokens": 0}

                agg = aggregated[model]
                agg["runs"] += stats.get("runs", 0)
                agg["failures"] += stats.get("failures", 0)
                agg["total_time"] += stats.get("total_time", 0)
                # Estimate total tokens from avg
                agg["total_tokens"] += stats.get("avg_tokens", 0) * stats.get("runs", 0)
        except (json.JSONDecodeError, KeyError):
            pass

    # Calculate averages
    for model, agg in aggregated.items():
        runs = agg["runs"]
        if runs > 0:
            agg["avg_time"] = round(agg["total_time"] / runs, 2)
            agg["avg_tokens"] = round(agg["total_tokens"] / runs)
            agg["failure_rate"] = round(agg["failures"] / (runs + agg["failures"]) * 100, 1)

    return aggregated


if __name__ == "__main__":
    import sys

    if len(sys.argv) < 2:
        print("Usage: triage_logger.py <command>")
        print("Commands: list, cleanup, benchmark")
        sys.exit(1)

    cmd = sys.argv[1]

    if cmd == "list":
        runs = list_runs()
        print(f"Found {len(runs)} triage runs:\n")
        for r in runs[:10]:
            print(f"  {r['run_id']} - {r['issues']} issues, {r['duration']/60:.1f}min")

    elif cmd == "cleanup":
        result = cleanup_old_data()
        print(f"Cleanup: deleted {result['deleted']} dirs, compressed {result['compressed']} logs")

    elif cmd == "benchmark":
        benchmarks = get_model_benchmarks()
        print("Model Benchmarks (all runs):\n")
        print(f"{'Model':<40} {'Runs':>6} {'Avg Time':>10} {'Fail %':>8}")
        print("-" * 70)
        for model, stats in sorted(benchmarks.items(), key=lambda x: x[1].get("avg_time", 0), reverse=True):
            print(f"{model:<40} {stats['runs']:>6} {stats.get('avg_time', 0):>9.1f}s {stats.get('failure_rate', 0):>7.1f}%")
