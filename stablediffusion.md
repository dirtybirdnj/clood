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

#!/bin/bash

# 1. Force Homebrew Python 3.10
# This matches the path installed by 'brew install python@3.10'
export python_cmd="/opt/homebrew/opt/python@3.10/libexec/bin/python"

# 2. Optimization Flags for M4 / Metal
# --no-half: Essential. Prevents black images/NaN errors on Metal.
# --no-half-vae: Fixes artifacts during the decoding phase.
# --opt-sub-quad-attention: Significantly reduces memory usage.
# --skip-torch-cuda-test: Stops the script from looking for NVIDIA drivers.
export COMMANDLINE_ARGS="--no-half --no-half-vae --opt-sub-quad-attention --skip-torch-cuda-test"

# 3. Updates
# Set to "true" if you want it to git pull every time you launch.
export GIT_PULL="false"

# 4. Launch
# First run will take ~10 mins to download PyTorch and the base model.
./webui.sh