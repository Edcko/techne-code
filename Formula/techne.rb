class Techne < Formula
  desc "AI-powered coding agent with TUI interface"
  homepage "https://github.com/Edcko/techne-code"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Edcko/techne-code/releases/download/v#{version}/techne_#{version}_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER_DARWIN_ARM64_SHA256"
    else
      url "https://github.com/Edcko/techne-code/releases/download/v#{version}/techne_#{version}_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER_DARWIN_AMD64_SHA256"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/Edcko/techne-code/releases/download/v#{version}/techne_#{version}_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_ARM64_SHA256"
    else
      url "https://github.com/Edcko/techne-code/releases/download/v#{version}/techne_#{version}_linux_amd64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_AMD64_SHA256"
    end
  end

  def install
    bin.install "techne"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/techne version")
  end
end
