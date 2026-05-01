class Onemcp < Formula
  desc "Local-first MCP hub, router, and CLI for teams and individuals"
  homepage "https://1mcp.in"
  license "Apache-2.0"
  version "0.3.3"

  on_macos do
    on_arm do
      url "https://github.com/SaiAvinashPatoju/1mcp.in/releases/download/v0.3.3/mach1-darwin-arm64.tar.gz"
      sha256 "e96eda566fc7770a572bedb6fb03f274b5a02fd585bf3ffe8a31d10cea2453b0"
    end
    on_intel do
      url "https://github.com/SaiAvinashPatoju/1mcp.in/releases/download/v0.3.3/mach1-darwin-amd64.tar.gz"
      sha256 "511ed7d5bd2d095662394ca39bb514178879dfbe6218acb8bad74228a4dbdbc9"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/SaiAvinashPatoju/1mcp.in/releases/download/v0.3.3/mach1-linux-amd64.tar.gz"
      sha256 "eae728ad011aeaabfc44a2cd0454583ab5ddd67bb8d8ddc658aea0f5911f87de"
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
