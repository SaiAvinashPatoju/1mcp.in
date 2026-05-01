class Onemcp < Formula
  desc "Local-first MCP hub, router, and CLI for teams and individuals"
  homepage "https://1mcp.in"
  license "Apache-2.0"
  version "0.3.4"

  on_macos do
    on_arm do
      url "https://github.com/SaiAvinashPatoju/1mcp.in/releases/download/v0.3.4/mach1-darwin-arm64.tar.gz"
    end
    on_intel do
      url "https://github.com/SaiAvinashPatoju/1mcp.in/releases/download/v0.3.4/mach1-darwin-amd64.tar.gz"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/SaiAvinashPatoju/1mcp.in/releases/download/v0.3.4/mach1-linux-amd64.tar.gz"
    end
  end

  def install
    bin.install "mach1"
    bin.install "mach1ctl"
    man1.install Dir["man/*.1"] if Dir.exist?("man")
  end

  def post_install
    ohai "1mcp.in is installed!"
    ohai "Run `mach1ctl start` to launch the router."
    ohai "Run `mach1ctl connect vscode` to connect VS Code."
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/mach1ctl doctor 2>&1 || true")
  end
end
