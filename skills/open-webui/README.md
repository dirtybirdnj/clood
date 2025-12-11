# open-webui Tools & Functions

Python tools and functions for open-webui. Import via Admin Panel > Workspace > Tools.

## Adding a new tool

1. Save the Python file here with a descriptive name
2. In open-webui: Admin Panel > Workspace > Tools > Create
3. Paste the code and configure

## Exporting existing tools

```bash
# Copy database from container
docker cp open-webui:/app/backend/data/webui.db /tmp/webui.db

# Run export script
cd scripts
python3 export-openwebui-tools.py /tmp/webui.db ../skills/open-webui/
```

## Tool Template

```python
"""
title: My Tool Name
author: you
version: 0.1.0
"""

from pydantic import BaseModel, Field
from typing import Optional

class Tools:
    def __init__(self):
        pass

    def my_function(self, param: str) -> str:
        """
        Description of what this tool does.

        :param param: Description of parameter
        :return: What it returns
        """
        # Your code here
        return f"Result: {param}"
```
