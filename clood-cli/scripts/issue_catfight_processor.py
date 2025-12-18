#!/usr/bin/env python3
"""Issue Catfight Processor - Process GitHub issues through model gauntlet

Features:
- Label-aware: skips already-triaged issues
- Auto-labels: adds scope and consensus labels based on model outputs
- Resume-capable: can restart and pick up where it left off
"""

import json
import subprocess
import os
import sys
import re
import time
from datetime import datetime
from collections import Counter

def log(msg):
    print(f'[{datetime.now().strftime("%H:%M:%S")}] {msg}', flush=True)

def check_ollama():
    """Health check - make sure ollama is responding"""
    try:
        result = subprocess.run(['curl', '-s', 'http://localhost:11434/api/tags'],
                              capture_output=True, timeout=10)
        return result.returncode == 0
    except:
        return False

def get_issue_labels(issue):
    """Extract label names from issue data"""
    return [l['name'] for l in issue.get('labels', [])]

def should_process(issue, hostname_tag):
    """Determine if issue should be processed based on labels"""
    labels = get_issue_labels(issue)

    # Skip if already triaged by this machine (unless needs-retriage)
    if hostname_tag in labels and 'needs-retriage' not in labels:
        return False, "already triaged by this machine"

    # Skip if fully triaged (has generic 'triaged' label) unless needs-retriage
    if 'triaged' in labels and 'needs-retriage' not in labels:
        return False, "already triaged"

    # Skip if marked as triage-failed (needs manual intervention)
    # Actually, we might want to retry these - comment out for now
    # if 'triage-failed' in labels:
    #     return False, "previously failed - needs manual review"

    return True, "ready to process"

def extract_size_estimates(results_text):
    """Parse size estimates from model outputs

    Looks for patterns like:
    - **Size:** XS
    - **Size:** S
    - Size: M
    """
    sizes = []
    # Match various formats models might use
    patterns = [
        r'\*\*Size:\*\*\s*(XS|S|M|L|XL)',
        r'Size:\s*(XS|S|M|L|XL)',
        r'\bSize\b[:\s]*(XS|S|M|L|XL)',
    ]

    for pattern in patterns:
        matches = re.findall(pattern, results_text, re.IGNORECASE)
        sizes.extend([s.upper() for s in matches])

    return sizes

def extract_open_questions(results_text):
    """Check if models identified open questions or unclear requirements"""
    indicators = [
        'open question',
        'unclear',
        'need more information',
        'needs clarification',
        'ambiguous',
        'not specified',
        'missing requirement',
    ]

    text_lower = results_text.lower()
    return any(ind in text_lower for ind in indicators)

def analyze_results(results_text):
    """Analyze catfight results and determine labels to add

    Returns: (labels, consensus_summary)
    """
    labels = []
    consensus_summary = ""

    # Extract and analyze size estimates
    sizes = extract_size_estimates(results_text)
    if sizes:
        size_counts = Counter(sizes)
        most_common_size, most_common_count = size_counts.most_common(1)[0]
        total_sizes = len(sizes)

        # Build consensus summary
        consensus_summary = f"{most_common_count}/{total_sizes} models agree: Size {most_common_size}"

        # If majority (>50%) agree on a size, use that
        if most_common_count > total_sizes / 2:
            labels.append(f'scope:{most_common_size}')
        else:
            labels.append('scope-disputed')
            consensus_summary = f"⚠️ DISPUTED - {size_counts.most_common()}"

    # Check for open questions
    if extract_open_questions(results_text):
        labels.append('needs-clarification')
    else:
        labels.append('actionable')

    return labels, consensus_summary

def add_labels(issue_num, labels, repo):
    """Add labels to a GitHub issue"""
    for label in labels:
        cmd = f'gh issue edit {issue_num} --repo {repo} --add-label "{label}"'
        try:
            subprocess.run(cmd, shell=True, capture_output=True, timeout=30)
        except Exception as e:
            log(f'    ⚠ Failed to add label {label}: {e}')

def remove_label(issue_num, label, repo):
    """Remove a label from a GitHub issue"""
    cmd = f'gh issue edit {issue_num} --repo {repo} --remove-label "{label}"'
    try:
        subprocess.run(cmd, shell=True, capture_output=True, timeout=30)
    except:
        pass  # Label might not exist, that's fine

def main():
    # Configuration from environment or defaults
    issues_file = os.environ.get('ISSUES_FILE', '/tmp/catfight-issues.json')
    log_dir = os.environ.get('LOG_DIR', '/tmp/catfight-triage')
    models = os.environ.get('MODELS', 'tinyllama')
    repo = os.environ.get('REPO', 'dirtybirdnj/clood')
    hostname = os.environ.get('HOSTNAME', 'unknown')
    clood_path = os.environ.get('CLOOD_PATH', './clood')

    # Determine hostname tag for labels
    hostname_lower = hostname.lower()
    if 'mini' in hostname_lower:
        hostname_tag = 'triaged:mini'
    elif 'macbook' in hostname_lower or 'laptop' in hostname_lower:
        hostname_tag = 'triaged:laptop'
    else:
        hostname_tag = 'triaged'

    log(f'Hostname tag: {hostname_tag}')

    # Read issues
    with open(issues_file, 'r') as f:
        issues = json.load(f)

    prompt_template = '''You are analyzing GitHub Issue #{number} from the clood project (a CLI tool for orchestrating local LLM inference across a server garden).

## ISSUE
**Title:** {title}

**Body:**
{body}

---

## YOUR TASK
Analyze this issue and provide a structured response that will be posted as a GitHub comment.

Format your response EXACTLY like this:

## 🤖 Model Analysis

### Understanding
[2-3 sentence summary of what is being requested]

### Scope Estimate
**Size:** [XS/S/M/L/XL]
**Reasoning:** [Why this size]

### Implementation Plan
1. [Step 1]
2. [Step 2]
3. [etc]

### Code Sketch
```[language]
[Key code or pseudocode if applicable, otherwise write "N/A - planning/design issue"]
```

### Open Questions
- [Question 1]
- [Question 2]

---
*Generated by clood catfight triage*
'''

    processed = 0
    skipped = 0
    failed = 0

    for i, issue in enumerate(issues):
        num = issue['number']
        title = issue['title']
        body = issue['body'] or 'No description provided.'
        labels = get_issue_labels(issue)

        log(f'\n[{i+1}/{len(issues)}] Issue #{num}: {title[:50]}...')

        # Check if we should process this issue
        should, reason = should_process(issue, hostname_tag)
        if not should:
            log(f'  ⏭ Skipping - {reason}')
            skipped += 1
            continue

        # De-icing: Check ollama is alive before each issue
        if not check_ollama():
            log('  ⚠ Ollama not responding, waiting 30s...')
            time.sleep(30)
            if not check_ollama():
                log('  ✗ Ollama still down, skipping issue')
                add_labels(num, ['triage-failed'], repo)
                failed += 1
                continue

        # Create prompt
        prompt = prompt_template.format(
            number=num,
            title=title,
            body=body
        )

        # Save prompt
        prompt_file = f'{log_dir}/issue-{num}-prompt.txt'
        with open(prompt_file, 'w') as f:
            f.write(prompt)

        # Run catfight
        result_file = f'{log_dir}/issue-{num}-results.txt'
        cmd = f'{clood_path} catfight -m "{models}" -f "{prompt_file}" > "{result_file}" 2>&1'

        try:
            subprocess.run(cmd, shell=True, timeout=1800)  # 30 min timeout
            log('  ✓ Catfight complete')

            # Read results
            with open(result_file, 'r') as f:
                results = f.read()

            log(f'  ✓ Results saved to {result_file}')

            # Analyze results and determine labels
            result_labels, consensus_summary = analyze_results(results)
            result_labels.append(hostname_tag)  # Mark as triaged by this machine

            log(f'  📋 Labels to add: {result_labels}')
            if consensus_summary:
                log(f'  📊 Consensus: {consensus_summary}')

            # Truncate results if too long
            max_len = 12000
            if len(results) > max_len:
                results = results[-max_len:]

            model_count = len(models.split(','))

            # Build consensus line
            consensus_line = f"\n**📊 Consensus:** {consensus_summary}\n" if consensus_summary else ""

            # Build comment
            comment = f'''## 🐱 Catfight Triage Results ({hostname})

This issue was analyzed by the clood model gauntlet running on **{hostname}**.

**Models ({model_count} cats):**
```
{models}
```
{consensus_line}
<details>
<summary>Click to expand full analysis</summary>

```
{results}
```

</details>

---
*Automated triage by clood catfight on {hostname} - {datetime.now().strftime("%Y-%m-%d %H:%M")}*
'''

            # Post to GitHub
            comment_file = f'{log_dir}/issue-{num}-comment.md'
            with open(comment_file, 'w') as f:
                f.write(comment)

            post_cmd = f'gh issue comment {num} --repo {repo} --body-file "{comment_file}"'
            subprocess.run(post_cmd, shell=True)
            log(f'  ✓ Comment posted to Issue #{num}')

            # Add labels
            add_labels(num, result_labels, repo)

            # Remove needs-retriage if it was set
            if 'needs-retriage' in labels:
                remove_label(num, 'needs-retriage', repo)

            log(f'  ✓ Labels updated')
            processed += 1

            # De-icing: Sleep between issues to avoid GitHub rate limiting
            log('  💤 Sleeping 10s before next issue...')
            time.sleep(10)

        except subprocess.TimeoutExpired:
            log(f'  ✗ Timeout on Issue #{num} (30 min exceeded)')
            add_labels(num, ['triage-failed'], repo)
            failed += 1
        except Exception as e:
            log(f'  ✗ Error on Issue #{num}: {e}')
            add_labels(num, ['triage-failed'], repo)
            failed += 1

    print('\n' + '='*50)
    print(f'🏁 CATFIGHT TRIAGE COMPLETE - {hostname}')
    print('='*50)
    print(f'  Processed: {processed}')
    print(f'  Skipped:   {skipped}')
    print(f'  Failed:    {failed}')
    print(f'  Total:     {len(issues)}')
    print(f'\nResults saved to: {log_dir}')

if __name__ == '__main__':
    main()
