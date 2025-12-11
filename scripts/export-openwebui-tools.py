#!/usr/bin/env python3
"""
Export tools and functions from open-webui SQLite database.
Run from host after copying webui.db from container:
  docker cp open-webui:/app/backend/data/webui.db /tmp/webui.db
  python3 export-openwebui-tools.py /tmp/webui.db ../skills/open-webui/
"""

import sqlite3
import json
import sys
import os
from pathlib import Path

def export_tools(db_path: str, output_dir: str):
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row

    output_path = Path(output_dir)
    output_path.mkdir(parents=True, exist_ok=True)

    # Export tools
    tools = conn.execute("SELECT * FROM tool").fetchall()
    print(f"Found {len(tools)} tools")

    for tool in tools:
        tool_dict = dict(tool)
        tool_id = tool_dict.get('id', 'unknown')

        # Save as JSON for full metadata
        json_path = output_path / f"{tool_id}.json"
        with open(json_path, 'w') as f:
            json.dump(tool_dict, f, indent=2, default=str)
        print(f"  Exported: {json_path}")

        # If there's content/code, save it separately as .py for readability
        content = tool_dict.get('content') or tool_dict.get('code')
        if content:
            py_path = output_path / f"{tool_id}.py"
            with open(py_path, 'w') as f:
                f.write(f"# Tool: {tool_dict.get('name', tool_id)}\n")
                f.write(f"# Exported from open-webui\n\n")
                f.write(content)
            print(f"  Exported: {py_path}")

    # Export functions
    functions = conn.execute("SELECT * FROM function").fetchall()
    print(f"Found {len(functions)} functions")

    for func in functions:
        func_dict = dict(func)
        func_id = func_dict.get('id', 'unknown')

        json_path = output_path / f"function_{func_id}.json"
        with open(json_path, 'w') as f:
            json.dump(func_dict, f, indent=2, default=str)
        print(f"  Exported: {json_path}")

        content = func_dict.get('content') or func_dict.get('code')
        if content:
            py_path = output_path / f"function_{func_id}.py"
            with open(py_path, 'w') as f:
                f.write(f"# Function: {func_dict.get('name', func_id)}\n")
                f.write(f"# Exported from open-webui\n\n")
                f.write(content)
            print(f"  Exported: {py_path}")

    conn.close()
    print(f"\nExport complete. Files saved to {output_path}")

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: export-openwebui-tools.py <webui.db path> <output dir>")
        print("Example: export-openwebui-tools.py /tmp/webui.db ../skills/open-webui/")
        sys.exit(1)

    export_tools(sys.argv[1], sys.argv[2])
