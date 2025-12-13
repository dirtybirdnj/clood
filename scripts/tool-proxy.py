#!/usr/bin/env python3
"""Proxy that injects tools into Ollama API requests."""
from http.server import HTTPServer, BaseHTTPRequestHandler
import json, urllib.request

OLLAMA = "http://localhost:11434"
TOOLS = [
    {"type": "function", "function": {
        "name": "execute_command", "description": "Run a shell command",
        "parameters": {"type": "object", "properties": {"command": {"type": "string"}}, "required": ["command"]}}},
    {"type": "function", "function": {
        "name": "read_file", "description": "Read a file from disk",
        "parameters": {"type": "object", "properties": {"path": {"type": "string"}}, "required": ["path"]}}},
    {"type": "function", "function": {
        "name": "web_search", "description": "Search the web",
        "parameters": {"type": "object", "properties": {"query": {"type": "string"}}, "required": ["query"]}}},
]

class Proxy(BaseHTTPRequestHandler):
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        if 'tools' not in body and self.path in ['/v1/chat/completions', '/api/chat']:
            body['tools'] = TOOLS
            print(f"[proxy] Injected {len(TOOLS)} tools")
        req = urllib.request.Request(OLLAMA + self.path, json.dumps(body).encode(),
            {'Content-Type': 'application/json'})
        try:
            with urllib.request.urlopen(req) as r:
                data = r.read()
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(data)
        except Exception as e:
            self.send_error(500, str(e))

    def do_GET(self):
        with urllib.request.urlopen(OLLAMA + self.path) as r:
            self.send_response(200)
            self.send_header('Content-Type', r.headers.get('Content-Type', 'application/json'))
            self.end_headers()
            self.wfile.write(r.read())

    def log_message(self, fmt, *args): print(f"[proxy] {args[0]}")

if __name__ == '__main__':
    print("Tool proxy on :11435 -> Ollama :11434")
    HTTPServer(('', 11435), Proxy).serve_forever()
