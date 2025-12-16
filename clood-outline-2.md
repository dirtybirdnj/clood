The contents of `clood-outline-1.md` are:
```
Step 1: Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

Step 2: Install Crush
macOS: brew install charmbracelet/tap/crush
Linux: download from https://github.com/charmbracelet/crush/releases

Step 3: Pull a model
ollama pull llama3-groq-tool-use:8b

Step 4: Start SearXNG (for web search)
cd infrastructure && docker compose up -d searxng

Step 5: Copy Crush config and run Crush
mkdir -p ~/.config/crush
cp infrastructure/configs/crush/crush.json ~/.config/crush/crush.json
crush
```
The full implementation for the five steps is provided in the project's script directory.
