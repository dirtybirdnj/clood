"""
title: Code Directory Reader
author: mgilbert
version: 0.1.0
description: Read files from the mounted /app/code directory
"""

import os
from typing import Optional
from pydantic import BaseModel, Field


class Tools:
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

    def list_directory(self, path: str = "") -> str:
        """
        List files and directories in the code directory.

        :param path: Relative path within /app/code (empty string for root)
        :return: List of files and directories
        """
        full_path = os.path.join(self.valves.base_path, path)

        # Security check - prevent directory traversal
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

    def read_file(self, path: str) -> str:
        """
        Read the contents of a file from the code directory.

        :param path: Relative path to the file within /app/code
        :return: File contents or error message
        """
        full_path = os.path.join(self.valves.base_path, path)

        # Security check - prevent directory traversal
        real_path = os.path.realpath(full_path)
        if not real_path.startswith(os.path.realpath(self.valves.base_path)):
            return "Error: Access denied - path outside allowed directory"

        if not os.path.exists(real_path):
            return f"Error: File does not exist: {path}"

        if os.path.isdir(real_path):
            return f"Error: Path is a directory, not a file: {path}"

        file_size = os.path.getsize(real_path)
        if file_size > self.valves.max_file_size:
            return f"Error: File too large ({file_size} bytes). Max allowed: {self.valves.max_file_size} bytes"

        try:
            with open(real_path, 'r', encoding='utf-8', errors='replace') as f:
                content = f.read()
            return f"Contents of '{path}':\n\n{content}"
        except Exception as e:
            return f"Error reading file: {str(e)}"

    def search_files(self, pattern: str, path: str = "") -> str:
        """
        Search for files matching a pattern in the code directory.

        :param pattern: Substring to search for in filenames
        :param path: Relative path to search within (empty for entire code directory)
        :return: List of matching file paths
        """
        full_path = os.path.join(self.valves.base_path, path)

        # Security check
        real_path = os.path.realpath(full_path)
        if not real_path.startswith(os.path.realpath(self.valves.base_path)):
            return "Error: Access denied - path outside allowed directory"

        if not os.path.exists(real_path):
            return f"Error: Path does not exist: {path}"

        matches = []
        try:
            for root, dirs, files in os.walk(real_path):
                # Skip hidden directories
                dirs[:] = [d for d in dirs if not d.startswith('.')]

                for filename in files:
                    if pattern.lower() in filename.lower():
                        rel_path = os.path.relpath(
                            os.path.join(root, filename),
                            self.valves.base_path
                        )
                        matches.append(rel_path)

                if len(matches) >= 50:
                    matches.append("... (truncated, more than 50 matches)")
                    break

            if not matches:
                return f"No files found matching '{pattern}'"

            return f"Files matching '{pattern}':\n" + "\n".join(matches)
        except Exception as e:
            return f"Error searching: {str(e)}"

    def grep_in_files(self, search_text: str, file_extension: str = "", path: str = "") -> str:
        """
        Search for text content within files.

        :param search_text: Text to search for in file contents
        :param file_extension: Optional file extension filter (e.g., '.py', '.rs')
        :param path: Relative path to search within
        :return: Files and lines containing the search text
        """
        full_path = os.path.join(self.valves.base_path, path)

        # Security check
        real_path = os.path.realpath(full_path)
        if not real_path.startswith(os.path.realpath(self.valves.base_path)):
            return "Error: Access denied - path outside allowed directory"

        results = []
        files_searched = 0
        max_results = 30

        try:
            for root, dirs, files in os.walk(real_path):
                dirs[:] = [d for d in dirs if not d.startswith('.')]

                for filename in files:
                    if file_extension and not filename.endswith(file_extension):
                        continue

                    file_path = os.path.join(root, filename)

                    # Skip large files and binary files
                    try:
                        if os.path.getsize(file_path) > 50000:
                            continue

                        with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
                            for line_num, line in enumerate(f, 1):
                                if search_text.lower() in line.lower():
                                    rel_path = os.path.relpath(file_path, self.valves.base_path)
                                    results.append(f"{rel_path}:{line_num}: {line.strip()[:100]}")

                                    if len(results) >= max_results:
                                        results.append(f"... (truncated at {max_results} results)")
                                        return "\n".join(results)

                        files_searched += 1
                    except:
                        continue

            if not results:
                return f"No matches found for '{search_text}' (searched {files_searched} files)"

            return f"Found {len(results)} matches:\n" + "\n".join(results)
        except Exception as e:
            return f"Error: {str(e)}"
