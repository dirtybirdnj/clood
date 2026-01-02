"""
title: Clood Tools
author: clood
version: 0.1.0
description: MCP-like tools for codebase exploration via clood CLI
"""

import subprocess
import os
from typing import Callable, Any


class Tools:
    def __init__(self):
        self.valves = self.Valves()

    class Valves:
        """Configuration for clood tools"""
        CLOOD_PATH: str = "clood"
        WORKING_DIR: str = os.getcwd()

    def _run_clood(self, *args) -> str:
        """Run a clood command and return output"""
        try:
            result = subprocess.run(
                [self.valves.CLOOD_PATH] + list(args),
                cwd=self.valves.WORKING_DIR,
                capture_output=True,
                text=True,
                timeout=30
            )
            return result.stdout or result.stderr or "No output"
        except subprocess.TimeoutExpired:
            return "Command timed out after 30 seconds"
        except Exception as e:
            return f"Error: {e}"

    def grep(self, pattern: str, __event_emitter__: Callable[[dict], Any] = None) -> str:
        """
        Search the codebase for a pattern using regex.

        :param pattern: The regex pattern to search for
        :return: List of matching files and content
        """
        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": f"Searching for: {pattern}", "done": False}})

        result = self._run_clood("grep", pattern)

        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": "Search complete", "done": True}})

        return result

    def tree(self, path: str = ".", __event_emitter__: Callable[[dict], Any] = None) -> str:
        """
        Show directory structure of the codebase.

        :param path: The path to show tree for (default: current directory)
        :return: ASCII tree representation
        """
        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": f"Getting tree for: {path}", "done": False}})

        result = self._run_clood("tree", path)

        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": "Tree complete", "done": True}})

        return result

    def symbols(self, path: str = ".", __event_emitter__: Callable[[dict], Any] = None) -> str:
        """
        Extract symbols (functions, types, classes) from code files.

        :param path: The path to analyze
        :return: List of symbols with their locations
        """
        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": f"Extracting symbols from: {path}", "done": False}})

        result = self._run_clood("symbols", path)

        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": "Symbols extracted", "done": True}})

        return result

    def imports(self, file_path: str, __event_emitter__: Callable[[dict], Any] = None) -> str:
        """
        Analyze imports/dependencies of a file.

        :param file_path: The file to analyze
        :return: Import analysis
        """
        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": f"Analyzing imports for: {file_path}", "done": False}})

        result = self._run_clood("imports", file_path)

        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": "Analysis complete", "done": True}})

        return result

    def context(self, path: str = ".", __event_emitter__: Callable[[dict], Any] = None) -> str:
        """
        Generate LLM-optimized context summary for a project.

        :param path: The project path
        :return: Context summary
        """
        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": f"Generating context for: {path}", "done": False}})

        result = self._run_clood("context", path)

        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": "Context generated", "done": True}})

        return result

    def ask(self, question: str, model: str = "", __event_emitter__: Callable[[dict], Any] = None) -> str:
        """
        Query a local LLM with a question (inception - query another model mid-conversation).

        :param question: The question to ask
        :param model: Optional specific model to use
        :return: The model's response
        """
        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": f"Querying: {question[:50]}...", "done": False}})

        args = ["ask", question]
        if model:
            args.extend(["--model", model])

        result = self._run_clood(*args)

        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": "Query complete", "done": True}})

        return result

    def hosts(self, __event_emitter__: Callable[[dict], Any] = None) -> str:
        """
        List all Ollama hosts and their status.

        :return: Host status information
        """
        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": "Checking hosts...", "done": False}})

        result = self._run_clood("hosts")

        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": "Hosts checked", "done": True}})

        return result

    def models(self, __event_emitter__: Callable[[dict], Any] = None) -> str:
        """
        List all available models across all hosts.

        :return: Model list with host information
        """
        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": "Listing models...", "done": False}})

        result = self._run_clood("models")

        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": "Models listed", "done": True}})

        return result

    def git_diff(self, ref: str = "HEAD", __event_emitter__: Callable[[dict], Any] = None) -> str:
        """
        Show git diff for pending changes or between refs.

        :param ref: Git reference (default: HEAD for uncommitted changes)
        :return: Git diff output
        """
        result = subprocess.run(
            ["git", "diff", ref],
            cwd=self.valves.WORKING_DIR,
            capture_output=True,
            text=True,
            timeout=30
        )
        return result.stdout or result.stderr or "No changes"

    def git_log(self, count: int = 10, __event_emitter__: Callable[[dict], Any] = None) -> str:
        """
        Show recent git commits.

        :param count: Number of commits to show
        :return: Git log output
        """
        result = subprocess.run(
            ["git", "log", f"-{count}", "--oneline"],
            cwd=self.valves.WORKING_DIR,
            capture_output=True,
            text=True,
            timeout=30
        )
        return result.stdout or result.stderr or "No commits"
