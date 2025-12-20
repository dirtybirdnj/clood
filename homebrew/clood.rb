# Homebrew formula for clood
#
# To use this formula, create a tap:
#   1. Create repo: github.com/dirtybirdnj/homebrew-clood
#   2. Copy this file to Formula/clood.rb in that repo
#   3. Users can then: brew tap dirtybirdnj/clood && brew install clood
#
# Or install directly from this repo:
#   brew install --HEAD dirtybirdnj/clood/clood

class Clood < Formula
  desc "Lightning in a Bottle - Local LLM infrastructure for server gardens"
  homepage "https://github.com/dirtybirdnj/clood"
  license "MIT"
  head "https://github.com/dirtybirdnj/clood.git", branch: "main"

  # Uncomment when releases are available:
  # url "https://github.com/dirtybirdnj/clood/archive/refs/tags/v0.3.0.tar.gz"
  # sha256 "REPLACE_WITH_SHA256"
  # version "0.3.0"

  depends_on "go" => :build

  def install
    cd "clood-cli" do
      ldflags = %W[
        -s -w
        -X main.version=#{version}
        -X main.buildTime=#{Time.now.utc.iso8601}
      ]
      system "go", "build", *std_go_args(ldflags:), "./cmd/clood"
    end

    # Generate shell completions
    generate_completions_from_executable(bin/"clood", "completion")
  end

  def caveats
    <<~EOS
      To get started with clood:
        clood init          # Create default config
        clood preflight     # Check local capabilities
        clood hosts         # See available Ollama hosts

      For MCP integration with Claude Code, add to ~/.claude.json:
        {
          "mcpServers": {
            "clood": {
              "command": "#{bin}/clood",
              "args": ["mcp"]
            }
          }
        }

      Documentation: https://github.com/dirtybirdnj/clood
    EOS
  end

  test do
    assert_match "clood", shell_output("#{bin}/clood --version")
    assert_match "system", shell_output("#{bin}/clood --help")
  end
end
