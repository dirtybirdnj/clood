# Stable Diffusion Setup (M4 Apple Silicon)

**Prerequisites:** Homebrew installed.
**Instructions:** Run the code block below in your Terminal. It installs dependencies, clones the repo, generates the correct M4 configuration file, and launches the app.

```bash
# 1. Install System Dependencies (Python 3.10 is required)
brew install cmake protobuf rust python@3.10 git wget

# 2. Clone the Repository
cd ~
git clone [https://github.com/AUTOMATIC1111/stable-diffusion-webui.git](https://github.com/AUTOMATIC1111/stable-diffusion-webui.git)
cd stable-diffusion-webui

# 3. Create Configuration File (Optimized for M4/Metal)
# This creates/overwrites webui-user.sh with necessary flags to prevent black images/NaN errors
cat <<EOF > webui-user.sh
#!/bin/bash

# Force Homebrew Python 3.10
export python_cmd="/opt/homebrew/opt/python@3.10/libexec/bin/python"

# Optimization Flags:
# --no-half: Fixes black images on Metal
# --no-half-vae: Fixes VAE artifacts
# --opt-sub-quad-attention: Reduces memory usage
# --skip-torch-cuda-test: Bypasses NVIDIA check
export COMMANDLINE_ARGS="--no-half --no-half-vae --opt-sub-quad-attention --skip-torch-cuda-test"

# Set to "true" if you want auto-updates on launch
export GIT_PULL="false"
EOF

# 4. Launch
# First run will take ~10 mins to download PyTorch and the base model.
./webui.sh