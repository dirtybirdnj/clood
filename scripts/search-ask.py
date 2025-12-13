#!/usr/bin/env python3
"""Search SearXNG, feed results to Ollama, get answer."""
import sys, json, urllib.request, urllib.parse, argparse

SEARXNG_URL = "http://192.168.4.63:8888"
OLLAMA_URL = "http://localhost:11434"
MODEL = "llama3-groq-tool-use:8b"

def search(query: str, num_results: int = 5) -> list:
    """Search SearXNG and return results."""
    params = urllib.parse.urlencode({"q": query, "format": "json"})
    url = f"{SEARXNG_URL}/search?{params}"

    req = urllib.request.Request(url, headers={"User-Agent": "search-ask/1.0"})
    with urllib.request.urlopen(req, timeout=30) as r:
        data = json.loads(r.read())

    results = []
    for item in data.get("results", [])[:num_results]:
        results.append({
            "title": item.get("title", ""),
            "url": item.get("url", ""),
            "content": item.get("content", "")[:500]
        })
    return results

def ask(query: str, context: list, model: str = MODEL, ollama_url: str = None) -> str:
    """Ask Ollama with search context."""
    context_text = "\n\n".join([
        f"**{r['title']}**\n{r['url']}\n{r['content']}"
        for r in context
    ])

    prompt = f"""Answer the question using the search results below.

## Search Results
{context_text}

## Question
{query}

## Your Answer (cite sources with URLs)"""

    payload = {
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "stream": False
    }

    url = ollama_url or OLLAMA_URL
    req = urllib.request.Request(
        f"{url}/api/chat",
        json.dumps(payload).encode(),
        {"Content-Type": "application/json"}
    )

    with urllib.request.urlopen(req, timeout=300) as r:
        resp = json.loads(r.read())
        return resp.get("message", {}).get("content", "No response")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Search + Ask with local LLM")
    parser.add_argument("query", nargs="+", help="Your question")
    parser.add_argument("-n", "--num", type=int, default=5, help="Number of search results")
    parser.add_argument("-m", "--model", default=MODEL, help=f"Model (default: {MODEL})")
    parser.add_argument("-o", "--ollama", default=OLLAMA_URL, help="Ollama URL (default: localhost)")
    parser.add_argument("-s", "--search-only", action="store_true", help="Just show search results")
    args = parser.parse_args()

    OLLAMA_URL = args.ollama
    query = " ".join(args.query)
    print(f"🔍 Searching: {query}", file=sys.stderr)

    results = search(query, args.num)

    if args.search_only:
        for r in results:
            print(f"\n{r['title']}\n{r['url']}\n{r['content'][:200]}...")
    else:
        print(f"🤖 Asking {args.model} @ {args.ollama}...", file=sys.stderr)
        print(ask(query, results, args.model, args.ollama))
