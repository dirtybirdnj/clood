# clood
We want Claude! Too bad, you get clood. Setup tools for local hardware LLM.

This project serves as a storage mechanism for LLM Agent tools as well as a manifest of all the tools and setup steps to get LLMs running on an Ubuntu 25 workstation.

YMMV this literally "works on my machine" and it's only useful as a template to customize for your own needs. You have been warned.

1. use small ollama and qwen coding agents
2. xearch
3. there is a directory that has code checked out, instead of searching for the web I want to be able to manually manipulate these folders, or have clood do it for me. It's important that it has a place I can put files it can view without searching the internet. This allows me to download stuff myself and not get blocked / rate limited when claude does it.
4. It should expose open-webui to the local network, use a https://clood/ URL if possible
5. It should use a self-signed SSL cert, its only running on my local network.
6. Need to document all the CLI tools installed via apt, cargo, pip, any other tools.
7. Need to designate tools to run through a multi-pane tmux session: lazydocker, btop, a terminal session
   
