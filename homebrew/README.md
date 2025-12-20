# Homebrew Installation

## Quick Install (when tap is set up)

```bash
brew tap dirtybirdnj/clood
brew install clood
```

## Install from HEAD (development)

```bash
brew install --HEAD dirtybirdnj/clood/clood
```

## Setting Up the Tap

To create the Homebrew tap:

1. Create a new repo: `github.com/dirtybirdnj/homebrew-clood`
2. Copy `clood.rb` to `Formula/clood.rb` in that repo
3. Push to GitHub

Users can then install with:
```bash
brew tap dirtybirdnj/clood
brew install clood
```

## Updating the Formula

When creating a new release:

1. Tag the release: `git tag v0.3.0 && git push --tags`
2. Wait for GitHub Actions to create the release
3. Get the tarball SHA256:
   ```bash
   curl -sL https://github.com/dirtybirdnj/clood/archive/refs/tags/v0.3.0.tar.gz | shasum -a 256
   ```
4. Update `clood.rb` with new version and SHA256
5. Push to the tap repo

## Shell Completions

After installing, enable completions:

```bash
# Bash
echo 'source <(clood completion bash)' >> ~/.bashrc

# Zsh
echo 'source <(clood completion zsh)' >> ~/.zshrc

# Fish
clood completion fish > ~/.config/fish/completions/clood.fish
```
