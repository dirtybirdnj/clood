"""
title: Code Reader
author: clood
description: Read and explore code files from ~/Code (mounted at /app/code)
required_open_webui_version: 0.4.0
version: 0.2.0
license: MIT
"""

from pydantic import BaseModel, Field
from typing import Optional
from pathlib import Path
import os
import subprocess


class Tools:
    def __init__(self):
        self.valves = self.Valves()

    class Valves(BaseModel):
        CODE_DIR: str = Field(
            default="/app/code",
            description="Base directory for code (~/Code mounted here)"
        )

    def read_file(self, file_path: str) -> str:
        """
        Read the contents of a file.
        :param file_path: Path to the file relative to ~/Code (e.g., "clood/README.md")
        :return: File contents
        """
        full_path = os.path.join(self.valves.CODE_DIR, file_path)

        try:
            with open(full_path, 'r') as f:
                content = f.read()

            if len(content) > 50000:
                content = content[:50000] + "\n\n... [truncated, file too large]"

            return content
        except FileNotFoundError:
            return f"File not found: {file_path}"
        except Exception as e:
            return f"Error reading file: {e}"

    def list_files(self, path: str = "", pattern: str = "*") -> str:
        """
        List files in a directory.
        :param path: Directory path relative to ~/Code (e.g., "clood/internal")
        :param pattern: Glob pattern to filter files (e.g., "*.go", "*.py")
        :return: List of files
        """
        full_path = Path(self.valves.CODE_DIR) / path

        try:
            if not full_path.exists():
                return f"Directory not found: {path}"

            files = []
            for f in sorted(full_path.rglob(pattern)):
                rel_path = f.relative_to(full_path)
                if f.is_file():
                    size = f.stat().st_size
                    files.append(f"{rel_path} ({size} bytes)")
                else:
                    files.append(f"{rel_path}/")

            result = "\n".join(files[:100])
            if len(files) > 100:
                result += f"\n\n... and {len(files) - 100} more files"

            return result or "No files found"
        except Exception as e:
            return f"Error: {e}"

    def tree(self, path: str = "", max_depth: int = 3) -> str:
        """
        Show directory tree structure.
        :param path: Directory path relative to ~/Code
        :param max_depth: Maximum depth to show (default 3)
        :return: ASCII tree representation
        """
        full_path = Path(self.valves.CODE_DIR) / path

        if not full_path.exists():
            return f"Directory not found: {path}"

        lines = []

        def walk(dir_path: Path, prefix: str = "", depth: int = 0):
            if depth > max_depth:
                return

            try:
                entries = sorted(dir_path.iterdir(), key=lambda x: (not x.is_dir(), x.name.lower()))
            except PermissionError:
                return

            entries = [e for e in entries if not e.name.startswith('.')
                       and e.name not in ('node_modules', '__pycache__', 'vendor', 'dist', 'build')]

            for i, entry in enumerate(entries[:50]):
                is_last = i == len(entries) - 1 or i == 49
                connector = "└── " if is_last else "├── "

                if entry.is_dir():
                    lines.append(f"{prefix}{connector}{entry.name}/")
                    extension = "    " if is_last else "│   "
                    walk(entry, prefix + extension, depth + 1)
                else:
                    lines.append(f"{prefix}{connector}{entry.name}")

        lines.append(str(path or ".") + "/")
        walk(full_path)

        return "\n".join(lines[:500]) or "Empty directory"

    def search(self, pattern: str, path: str = "", file_pattern: str = "*") -> str:
        """
        Search for a pattern in files (like grep).
        :param pattern: Text pattern to search for
        :param path: Directory to search in relative to ~/Code
        :param file_pattern: File glob pattern (e.g., "*.go", "*.py")
        :return: Matching lines with file paths
        """
        full_path = Path(self.valves.CODE_DIR) / path

        if not full_path.exists():
            return f"Directory not found: {path}"

        results = []
        files_searched = 0

        for file_path in full_path.rglob(file_pattern):
            if not file_path.is_file():
                continue
            if any(skip in str(file_path) for skip in ['.git', 'node_modules', '__pycache__', 'vendor']):
                continue

            files_searched += 1
            if files_searched > 1000:
                results.append("\n... stopped after searching 1000 files")
                break

            try:
                with open(file_path, 'r', errors='ignore') as f:
                    for line_num, line in enumerate(f, 1):
                        if pattern.lower() in line.lower():
                            rel_path = file_path.relative_to(Path(self.valves.CODE_DIR))
                            results.append(f"{rel_path}:{line_num}: {line.rstrip()[:200]}")

                            if len(results) > 100:
                                results.append("\n... truncated after 100 matches")
                                return "\n".join(results)
            except:
                pass

        return "\n".join(results) if results else f"No matches for '{pattern}'"

    def git_status(self, path: str = "") -> str:
        """
        Show git status for a repository.
        :param path: Repository path relative to ~/Code
        :return: Git status output
        """
        full_path = os.path.join(self.valves.CODE_DIR, path)

        try:
            result = subprocess.run(
                ["git", "status", "--short"],
                cwd=full_path,
                capture_output=True,
                text=True,
                timeout=10
            )
            return result.stdout or "Clean working directory"
        except Exception as e:
            return f"Error: {e}"

    def git_log(self, path: str = "", count: int = 10) -> str:
        """
        Show recent git commits.
        :param path: Repository path relative to ~/Code
        :param count: Number of commits to show
        :return: Git log output
        """
        full_path = os.path.join(self.valves.CODE_DIR, path)

        try:
            result = subprocess.run(
                ["git", "log", f"-{count}", "--oneline"],
                cwd=full_path,
                capture_output=True,
                text=True,
                timeout=10
            )
            return result.stdout or "No commits"
        except Exception as e:
            return f"Error: {e}"

    def git_diff(self, path: str = "", file: str = "") -> str:
        """
        Show git diff for uncommitted changes.
        :param path: Repository path relative to ~/Code
        :param file: Optional specific file to diff
        :return: Git diff output
        """
        full_path = os.path.join(self.valves.CODE_DIR, path)

        try:
            cmd = ["git", "diff"]
            if file:
                cmd.append(file)

            result = subprocess.run(
                cmd,
                cwd=full_path,
                capture_output=True,
                text=True,
                timeout=10
            )
            diff = result.stdout
            if len(diff) > 20000:
                diff = diff[:20000] + "\n\n... [truncated]"
            return diff or "No changes"
        except Exception as e:
            return f"Error: {e}"
