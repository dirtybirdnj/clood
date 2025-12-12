# Last Session - 2025-12-11

## What We Did

### 1. Diagnosed Open WebUI Tool Calling Issues
- Discovered tool calling is **broken in Open WebUI v0.6.13+**
- Found GitHub issues confirming this: [#14577](https://github.com/open-webui/open-webui/issues/14577), [#14492](https://github.com/open-webui/open-webui/discussions/14492)
- Models would describe tools instead of calling them
- Web search was also broken - qwen was dumping entire prompts as search queries

### 2. Downgraded to v0.6.10
- Pinned Open WebUI to v0.6.10 (last version with working tools)
- Updated docker-compose.yml to use custom Dockerfile
- Created `infrastructure/Dockerfile.open-webui` with hf_xet pre-installed

### 3. Database Issues
- Had to wipe volumes multiple times due to schema incompatibility between versions
- Final clean state achieved

### 4. Infrastructure Changes
- Fixed network name: `webui-net` (not `clood-net`)
- Added hf_xet package to suppress HuggingFace download warnings
- Documented manual docker run command in README

## Current State

- **Open WebUI**: v0.6.10 running fresh (no data)
- **Volumes**: Wiped clean, new `open-webui-data` volume
- **Next step**: Create admin account, add Code Directory Reader tool, test tool calling

## Files Changed

- `README.md` - Added version pinning documentation and manual run command
- `infrastructure/docker-compose.yml` - Updated to build from Dockerfile, fixed network name
- `infrastructure/Dockerfile.open-webui` - New file, extends v0.6.10 with hf_xet

## To Resume

1. Go to http://localhost:3000
2. Create admin account
3. Go to Workspace → Tools → + Create
4. Paste contents of `skills/open-webui/code-directory-reader.py`
5. Save and test with qwen2.5-coder:7b

## Tool Code Location

The Code Directory Reader tool is at:
- `skills/open-webui/code-directory-reader.py` (Tool version - LLM calls it)
- `skills/open-webui/code-directory-reader-function.py` (Pipe version - acts as fake model)

## Known Issues

- Version display in admin panel may show wrong version (cached/db issue) - ignore it, container logs show correct version
- hf_xet needs to be installed after container creation (or build custom image)
