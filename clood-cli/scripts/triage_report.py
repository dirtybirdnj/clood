#!/usr/bin/env python3
"""Triage Report - Comprehensive post-mortem analysis and reporting

Features:
- Scope aggregation across all triaged issues
- Issue clustering by model consensus
- Response quality scoring
- GitHub label sync recommendations
"""

import json
import os
import re
import subprocess
import sys
from collections import defaultdict
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Tuple, Any

# Try to import from triage_analyze
try:
    from triage_analyze import parse_results_file, extract_scope_from_response
except ImportError:
    # Inline the functions if import fails
    def parse_results_file(filepath: Path) -> Dict:
        with open(filepath) as f:
            content = f.read()
        result = {"models": [], "winner": None, "responses": {}}
        pattern = r'>>> \[(\d+)/(\d+)\] (\S+) \(([^)]+)\)\s+(?:DONE (\d+\.?\d*)s \| (\d+) tokens \| (\d+\.?\d*) tok/s|(FAILED|ERROR)[^\n]*)'
        for match in re.finditer(pattern, content):
            idx, total, name, model, time, tokens, toks, error = match.groups()
            model_result = {"name": name, "model": model, "status": "done" if time else "failed"}
            if time:
                model_result["time"] = float(time)
                model_result["tokens"] = int(tokens)
                model_result["toks"] = float(toks)
            result["models"].append(model_result)
        winner_match = re.search(r'WINNER: (\S+) wins with ([\d.]+)s', content)
        if winner_match:
            result["winner"] = {"name": winner_match.group(1), "time": float(winner_match.group(2))}
        response_pattern = r'### (\S+) \(([^)]+)\)\n-+\n(.*?)(?=\n### |\n$|\Z)'
        for match in re.finditer(response_pattern, content, re.DOTALL):
            name, model, response = match.groups()
            result["responses"][model] = response.strip()
        return result

    def extract_scope_from_response(response: str) -> Optional[str]:
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


class TriageReporter:
    """Comprehensive triage report generator"""

    def __init__(self, run_dir: Path):
        self.run_dir = Path(run_dir)
        self.issues: Dict[int, Dict] = {}
        self._parse_all_results()

    def _parse_all_results(self):
        """Parse all result files in the run directory"""
        for results_file in sorted(self.run_dir.glob("issue-*-results.txt")):
            try:
                issue_num = int(re.search(r'issue-(\d+)-results', results_file.name).group(1))
                results = parse_results_file(results_file)

                # Extract scopes from each model
                scopes = {}
                for model, response in results.get("responses", {}).items():
                    scope = extract_scope_from_response(response)
                    if scope:
                        scopes[model] = scope

                # Calculate consensus
                if scopes:
                    scope_counts = defaultdict(int)
                    for s in scopes.values():
                        scope_counts[s] += 1
                    top_scope = max(scope_counts.items(), key=lambda x: x[1])
                    consensus_level = top_scope[1] / len(scopes)
                else:
                    top_scope = (None, 0)
                    consensus_level = 0

                self.issues[issue_num] = {
                    "results": results,
                    "scopes": scopes,
                    "scope_consensus": top_scope[0],
                    "scope_votes": top_scope[1],
                    "total_votes": len(scopes),
                    "consensus_level": consensus_level,
                    "models_run": len(results["models"]),
                    "models_succeeded": sum(1 for m in results["models"] if m["status"] == "done")
                }
            except Exception as e:
                print(f"Error parsing {results_file}: {e}", file=sys.stderr)

    # =========================================================================
    # 2. SCOPE AGGREGATION REPORT
    # =========================================================================

    def scope_report(self, output_json: bool = False) -> Dict:
        """Generate scope aggregation report"""
        report = {
            "total_issues": len(self.issues),
            "issues_with_scope": 0,
            "distribution": defaultdict(list),
            "by_scope": {}
        }

        for issue_num, data in self.issues.items():
            scope = data["scope_consensus"]
            if scope:
                report["issues_with_scope"] += 1
                report["distribution"][scope].append(issue_num)

        # Summary by scope
        for scope in ["XS", "S", "M", "L", "XL"]:
            issues = report["distribution"].get(scope, [])
            report["by_scope"][scope] = {
                "count": len(issues),
                "issues": issues,
                "percentage": round(len(issues) / max(report["issues_with_scope"], 1) * 100, 1)
            }

        if output_json:
            return report

        # Print report
        print("=" * 60)
        print("SCOPE AGGREGATION REPORT")
        print("=" * 60)
        print(f"\nTotal Issues: {report['total_issues']}")
        print(f"Issues with Scope: {report['issues_with_scope']}")
        print(f"Missing Scope: {report['total_issues'] - report['issues_with_scope']}")
        print("\nDISTRIBUTION:")
        print("-" * 40)

        for scope in ["XS", "S", "M", "L", "XL"]:
            data = report["by_scope"][scope]
            bar = "█" * int(data["percentage"] / 5)
            print(f"  {scope:>2}: {bar:<20} {data['count']:>3} ({data['percentage']:.0f}%)")
            if data["issues"]:
                print(f"      Issues: {', '.join(f'#{i}' for i in sorted(data['issues'])[:10])}")
                if len(data["issues"]) > 10:
                    print(f"      ... and {len(data['issues']) - 10} more")

        print("\nPRIORITY QUEUE (by effort):")
        print("-" * 40)
        # XS first, then S, M, L, XL
        for scope in ["XS", "S", "M", "L", "XL"]:
            issues = report["distribution"].get(scope, [])
            if issues:
                print(f"  {scope}: {', '.join(f'#{i}' for i in sorted(issues))}")

        return report

    # =========================================================================
    # 3. ISSUE CLUSTERING BY MODEL AGREEMENT
    # =========================================================================

    def consensus_report(self, output_json: bool = False) -> Dict:
        """Cluster issues by model agreement level"""
        report = {
            "high_consensus": [],      # 75%+ agreement
            "medium_consensus": [],    # 50-74% agreement
            "low_consensus": [],       # 25-49% agreement
            "no_consensus": [],        # <25% or no votes
            "needs_review": []         # Failed models or issues
        }

        for issue_num, data in self.issues.items():
            level = data["consensus_level"]
            succeeded = data["models_succeeded"]
            total = data["models_run"]

            # Check for failures
            if succeeded < total * 0.75:
                report["needs_review"].append({
                    "issue": issue_num,
                    "reason": f"Only {succeeded}/{total} models succeeded",
                    "scope": data["scope_consensus"]
                })
            elif level >= 0.75:
                report["high_consensus"].append({
                    "issue": issue_num,
                    "scope": data["scope_consensus"],
                    "agreement": f"{data['scope_votes']}/{data['total_votes']}",
                    "level": round(level * 100)
                })
            elif level >= 0.5:
                report["medium_consensus"].append({
                    "issue": issue_num,
                    "scope": data["scope_consensus"],
                    "agreement": f"{data['scope_votes']}/{data['total_votes']}",
                    "level": round(level * 100),
                    "votes": data["scopes"]
                })
            elif level >= 0.25:
                report["low_consensus"].append({
                    "issue": issue_num,
                    "scope": data["scope_consensus"],
                    "votes": data["scopes"]
                })
            else:
                report["no_consensus"].append({
                    "issue": issue_num,
                    "votes": data["scopes"]
                })

        if output_json:
            return report

        # Print report
        print("=" * 60)
        print("ISSUE CLUSTERING BY MODEL CONSENSUS")
        print("=" * 60)

        print(f"\n✅ HIGH CONSENSUS (75%+): {len(report['high_consensus'])} issues")
        print("-" * 40)
        for item in sorted(report["high_consensus"], key=lambda x: x["level"], reverse=True)[:15]:
            print(f"  #{item['issue']:>3} - Scope: {item['scope']} ({item['agreement']} agree, {item['level']}%)")

        print(f"\n⚠️  MEDIUM CONSENSUS (50-74%): {len(report['medium_consensus'])} issues")
        print("-" * 40)
        for item in report["medium_consensus"][:10]:
            votes_str = ", ".join(f"{m}: {s}" for m, s in list(item.get("votes", {}).items())[:3])
            print(f"  #{item['issue']:>3} - Scope: {item['scope']} ({item['level']}%) - {votes_str}")

        print(f"\n🔶 LOW CONSENSUS (25-49%): {len(report['low_consensus'])} issues")
        print("-" * 40)
        for item in report["low_consensus"][:10]:
            votes_str = ", ".join(f"{m}: {s}" for m, s in list(item.get("votes", {}).items())[:3])
            print(f"  #{item['issue']:>3} - Split: {votes_str}")

        print(f"\n❌ NO CONSENSUS (<25%): {len(report['no_consensus'])} issues")
        print("-" * 40)
        for item in report["no_consensus"][:10]:
            print(f"  #{item['issue']:>3} - No clear scope")

        print(f"\n🔧 NEEDS REVIEW (failures): {len(report['needs_review'])} issues")
        print("-" * 40)
        for item in report["needs_review"][:10]:
            print(f"  #{item['issue']:>3} - {item['reason']}")

        return report

    # =========================================================================
    # 5. RESPONSE QUALITY SCORING
    # =========================================================================

    def quality_report(self, output_json: bool = False) -> Dict:
        """Score response quality across models and issues"""
        report = {
            "model_quality": defaultdict(lambda: {
                "has_code": 0, "has_plan": 0, "has_diagram": 0,
                "has_questions": 0, "total": 0, "avg_tokens": 0, "total_tokens": 0
            }),
            "issue_quality": {},
            "best_responses": [],
            "weak_responses": []
        }

        for issue_num, data in self.issues.items():
            issue_scores = []

            for model, response in data["results"].get("responses", {}).items():
                score = self._score_response(response)

                # Update model stats
                mq = report["model_quality"][model]
                mq["total"] += 1
                mq["has_code"] += 1 if score["has_code"] else 0
                mq["has_plan"] += 1 if score["has_plan"] else 0
                mq["has_diagram"] += 1 if score["has_diagram"] else 0
                mq["has_questions"] += 1 if score["has_questions"] else 0
                mq["total_tokens"] += score["tokens"]

                issue_scores.append({
                    "model": model,
                    "score": score["total_score"],
                    "tokens": score["tokens"],
                    "details": score
                })

                # Track best/weak
                if score["total_score"] >= 4:
                    report["best_responses"].append({
                        "issue": issue_num,
                        "model": model,
                        "score": score["total_score"],
                        "tokens": score["tokens"]
                    })
                elif score["total_score"] <= 1 and score["tokens"] > 50:
                    report["weak_responses"].append({
                        "issue": issue_num,
                        "model": model,
                        "score": score["total_score"],
                        "tokens": score["tokens"]
                    })

            report["issue_quality"][issue_num] = issue_scores

        # Calculate averages
        for model, mq in report["model_quality"].items():
            if mq["total"] > 0:
                mq["avg_tokens"] = round(mq["total_tokens"] / mq["total"])
                mq["code_rate"] = round(mq["has_code"] / mq["total"] * 100, 1)
                mq["plan_rate"] = round(mq["has_plan"] / mq["total"] * 100, 1)
                mq["diagram_rate"] = round(mq["has_diagram"] / mq["total"] * 100, 1)

        if output_json:
            return {
                "model_quality": dict(report["model_quality"]),
                "best_responses": report["best_responses"][:20],
                "weak_responses": report["weak_responses"][:20]
            }

        # Print report
        print("=" * 60)
        print("RESPONSE QUALITY SCORING")
        print("=" * 60)

        print("\nMODEL QUALITY SCORES:")
        print("-" * 70)
        print(f"{'Model':<30} {'Code%':>7} {'Plan%':>7} {'Diag%':>7} {'AvgTok':>8}")
        print("-" * 70)

        for model, mq in sorted(report["model_quality"].items(),
                                key=lambda x: x[1].get("code_rate", 0), reverse=True):
            print(f"{model:<30} {mq.get('code_rate', 0):>6.0f}% {mq.get('plan_rate', 0):>6.0f}% "
                  f"{mq.get('diagram_rate', 0):>6.0f}% {mq.get('avg_tokens', 0):>8}")

        print(f"\n⭐ BEST RESPONSES ({len(report['best_responses'])} total):")
        print("-" * 40)
        for item in sorted(report["best_responses"], key=lambda x: x["score"], reverse=True)[:10]:
            print(f"  #{item['issue']:>3} - {item['model']:<25} (score: {item['score']}, {item['tokens']} tok)")

        print(f"\n⚠️  WEAK RESPONSES ({len(report['weak_responses'])} total):")
        print("-" * 40)
        for item in report["weak_responses"][:10]:
            print(f"  #{item['issue']:>3} - {item['model']:<25} (score: {item['score']}, {item['tokens']} tok)")

        return report

    def _score_response(self, response: str) -> Dict:
        """Score a single response for quality indicators"""
        score = {
            "has_code": bool(re.search(r'```\w*\n', response)),
            "has_plan": bool(re.search(r'(implementation plan|step \d|phase \d|\d\.\s+\w)', response, re.I)),
            "has_diagram": bool(re.search(r'```mermaid', response, re.I)),
            "has_questions": bool(re.search(r'(open question|unclear|need.*(clarif|more info))', response, re.I)),
            "has_scope": extract_scope_from_response(response) is not None,
            "tokens": len(response.split())
        }

        # Calculate total score (0-5)
        score["total_score"] = sum([
            score["has_code"],
            score["has_plan"],
            score["has_diagram"],
            score["has_questions"],
            score["has_scope"]
        ])

        return score

    # =========================================================================
    # 6. GITHUB LABEL SYNC
    # =========================================================================

    def label_sync_report(self, output_json: bool = False, apply: bool = False) -> Dict:
        """Generate GitHub label recommendations"""
        report = {
            "labels_to_add": [],
            "labels_to_create": [],
            "issues_to_update": []
        }

        # Define label mappings
        scope_labels = {
            "XS": "scope:xs",
            "S": "scope:s",
            "M": "scope:m",
            "L": "scope:l",
            "XL": "scope:xl"
        }

        consensus_labels = {
            "high": "consensus:high",
            "medium": "consensus:medium",
            "low": "consensus:low",
            "none": "needs-scope-review"
        }

        # Required labels
        report["labels_to_create"] = list(scope_labels.values()) + list(consensus_labels.values()) + ["triaged"]

        for issue_num, data in self.issues.items():
            labels_to_add = ["triaged"]

            # Scope label
            scope = data["scope_consensus"]
            if scope and scope in scope_labels:
                labels_to_add.append(scope_labels[scope])

            # Consensus label
            level = data["consensus_level"]
            if level >= 0.75:
                labels_to_add.append(consensus_labels["high"])
            elif level >= 0.5:
                labels_to_add.append(consensus_labels["medium"])
            elif level >= 0.25:
                labels_to_add.append(consensus_labels["low"])
            else:
                labels_to_add.append(consensus_labels["none"])

            # Check for failures
            if data["models_succeeded"] < data["models_run"] * 0.75:
                labels_to_add.append("needs-retriage")

            report["issues_to_update"].append({
                "issue": issue_num,
                "add_labels": labels_to_add,
                "scope": scope,
                "consensus": round(level * 100)
            })

        if output_json:
            return report

        # Print report
        print("=" * 60)
        print("GITHUB LABEL SYNC RECOMMENDATIONS")
        print("=" * 60)

        print("\nLABELS TO CREATE (if not exist):")
        print("-" * 40)
        for label in report["labels_to_create"]:
            print(f"  - {label}")

        print(f"\nISSUES TO UPDATE ({len(report['issues_to_update'])}):")
        print("-" * 60)
        print(f"{'Issue':>6} {'Scope':>6} {'Consensus':>10} Labels")
        print("-" * 60)

        for item in sorted(report["issues_to_update"], key=lambda x: x["issue"]):
            labels = ", ".join(item["add_labels"])
            print(f"#{item['issue']:>5} {item['scope'] or '-':>6} {item['consensus']:>9}% {labels}")

        if apply:
            print("\nAPPLYING LABELS...")
            self._apply_labels(report)

        print("\nTo apply these labels, run:")
        print("  python3 triage_report.py label-sync --apply")

        return report

    def _apply_labels(self, report: Dict):
        """Actually apply labels to GitHub issues"""
        # First, ensure labels exist
        for label in report["labels_to_create"]:
            try:
                # Check if label exists, create if not
                result = subprocess.run(
                    ["gh", "label", "create", label, "--repo", "dirtybirdnj/clood", "--force"],
                    capture_output=True, text=True
                )
            except Exception as e:
                print(f"  Warning: Could not create label {label}: {e}")

        # Apply labels to issues
        for item in report["issues_to_update"]:
            try:
                labels = ",".join(item["add_labels"])
                result = subprocess.run(
                    ["gh", "issue", "edit", str(item["issue"]),
                     "--repo", "dirtybirdnj/clood",
                     "--add-label", labels],
                    capture_output=True, text=True
                )
                if result.returncode == 0:
                    print(f"  ✓ #{item['issue']}: {labels}")
                else:
                    print(f"  ✗ #{item['issue']}: {result.stderr}")
            except Exception as e:
                print(f"  ✗ #{item['issue']}: {e}")


def main():
    import argparse

    parser = argparse.ArgumentParser(description="Triage post-mortem reports")
    parser.add_argument("command", choices=["scope", "consensus", "quality", "label-sync", "all"],
                       help="Report type to generate")
    parser.add_argument("--run-dir", "-d", type=Path,
                       help="Path to triage run directory")
    parser.add_argument("--json", action="store_true",
                       help="Output as JSON")
    parser.add_argument("--apply", action="store_true",
                       help="Apply changes (for label-sync)")

    args = parser.parse_args()

    # Find run directory
    if not args.run_dir:
        tmp_runs = list(Path("/tmp").glob("catfight-triage-*"))
        if tmp_runs:
            args.run_dir = sorted(tmp_runs)[-1]
            print(f"Using: {args.run_dir}\n")
        else:
            print("Error: No --run-dir specified and no runs found in /tmp")
            sys.exit(1)

    reporter = TriageReporter(args.run_dir)

    if args.command == "scope":
        result = reporter.scope_report(output_json=args.json)
        if args.json:
            print(json.dumps(result, indent=2))

    elif args.command == "consensus":
        result = reporter.consensus_report(output_json=args.json)
        if args.json:
            print(json.dumps(result, indent=2))

    elif args.command == "quality":
        result = reporter.quality_report(output_json=args.json)
        if args.json:
            print(json.dumps(result, indent=2))

    elif args.command == "label-sync":
        result = reporter.label_sync_report(output_json=args.json, apply=args.apply)
        if args.json:
            print(json.dumps(result, indent=2))

    elif args.command == "all":
        reporter.scope_report()
        print("\n")
        reporter.consensus_report()
        print("\n")
        reporter.quality_report()
        print("\n")
        reporter.label_sync_report()


if __name__ == "__main__":
    main()
