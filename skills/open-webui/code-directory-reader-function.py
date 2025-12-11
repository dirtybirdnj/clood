"""
title: Code Directory Reader
author: mgilbert
version: 0.1.0
description: Read files from the mounted /app/code directory
"""

import os
from pydantic import BaseModel, Field
from typing import Optional, Callable, Awaitable, Any


class Pipe:
    """
    A pipe/function that reads files from /app/code directory.
    Use commands like: list:/path or read:/path/file.py or search:pattern or grep:text
    """

    class Valves(BaseModel):
        base_path: str = Field(
            default="/app/code",
            description="Base path for code directory"
        )
        max_file_size: int = Field(
            default=100000,
            description="Maximum file size to read in bytes"
        )

    def __init__(self):
        self.valves = self.Valves()

    def pipe(self, body: dict, __user__: Optional[dict] = None) -> str:
        """Process the input and execute file operations."""
        messages = body.get("messages", [])
        if not messages:
            return "No message provided"

        user_message = messages[-1].get("content", "").strip()

        # Parse command format: command:argument
        if ":" in user_message:
            cmd, arg = user_message.split(":", 1)
            cmd = cmd.lower().strip()
            arg = arg.strip()

            if cmd == "list":
                return self._list_directory(arg)
            elif cmd == "read":
                return self._read_file(arg)
            elif cmd == "search":
                return self._search_files(arg)
            elif cmd == "grep":
                return self._grep_in_files(arg)

        return f"""Code Directory Reader - Commands:
- list:<path> - List directory contents (e.g., list: or list:clood)
- read:<path> - Read a file (e.g., read:clood/README.md)
- search:<pattern> - Find files by name (e.g., search:.py)
- grep:<text> - Search text in files (e.g., grep:TODO)

Your input: {user_message}"""

    def _list_directory(self, path: str = "") -> str:
        full_path = os.path.join(self.valves.base_path, path)
        real_path = os.path.realpath(full_path)

        if not real_path.startswith(os.path.realpath(self.valves.base_path)):
            return "Error: Access denied - path outside allowed directory"

        if not os.path.exists(real_path):
            return f"Error: Path does not exist: {path}"

        if not os.path.isdir(real_path):
            return f"Error: Not a directory: {path}"

        try:
            entries = []
            for entry in sorted(os.listdir(real_path)):
                entry_path = os.path.join(real_path, entry)
                if os.path.isdir(entry_path):
                    entries.append(f"[DIR]  {entry}/")
                else:
                    size = os.path.getsize(entry_path)
                    entries.append(f"[FILE] {entry} ({size} bytes)")

            if not entries:
                return f"Directory '{path or '/'}' is empty"

            return f"Contents of '{path or '/'}':\n" + "\n".join(entries)
        except Exception as e:
            return f"Error listing directory: {str(e)}"

    def _read_file(self, path: str) -> str:
        full_path = os.path.join(self.valves.base_path, path)
        real_path = os.path.realpath(full_path)

        if not real_path.startswith(os.path.realpath(self.valves.base_path)):
            return "Error: Access denied - path outside allowed directory"

        if not os.path.exists(real_path):
            return f"Error: File does not exist: {path}"

        if os.path.isdir(real_path):
            return f"Error: Path is a directory, not a file: {path}"

        file_size = os.path.getsize(real_path)
        if file_size > self.valves.max_file_size:
            return f"Error: File too large ({file_size} bytes). Max: {self.valves.max_file_size}"

        try:
            with open(real_path, 'r', encoding='utf-8', errors='replace') as f:
                content = f.read()
            return f"Contents of '{path}':\n\n{content}"
        except Exception as e:
            return f"Error reading file: {str(e)}"

    def _search_files(self, pattern: str) -> str:
        matches = []
        try:
            for root, dirs, files in os.walk(self.valves.base_path):
                dirs[:] = [d for d in dirs if not d.startswith('.')]
                for filename in files:
                    if pattern.lower() in filename.lower():
                        rel_path = os.path.relpath(
                            os.path.join(root, filename),
                            self.valves.base_path
                        )
                        matches.append(rel_path)
                        if len(matches) >= 50:
                            return f"Files matching '{pattern}':\n" + "\n".join(matches) + "\n... (truncated)"

            if not matches:
                return f"No files found matching '{pattern}'"
            return f"Files matching '{pattern}':\n" + "\n".join(matches)
        except Exception as e:
            return f"Error searching: {str(e)}"

    def _grep_in_files(self, search_text: str) -> str:
        results = []
        try:
            for root, dirs, files in os.walk(self.valves.base_path):
                dirs[:] = [d for d in dirs if not d.startswith('.')]
                for filename in files:
                    file_path = os.path.join(root, filename)
                    try:
                        if os.path.getsize(file_path) > 50000:
                            continue
                        with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
                            for line_num, line in enumerate(f, 1):
                                if search_text.lower() in line.lower():
                                    rel_path = os.path.relpath(file_path, self.valves.base_path)
                                    results.append(f"{rel_path}:{line_num}: {line.strip()[:80]}")
                                    if len(results) >= 30:
                                        return f"Found {len(results)} matches:\n" + "\n".join(results)
                    except:
                        continue

            if not results:
                return f"No matches found for '{search_text}'"
            return f"Found {len(results)} matches:\n" + "\n".join(results)
        except Exception as e:
            return f"Error: {str(e)}"
