class Opsfile < Formula
  desc "Like make/Makefile but for live operations commands"
  homepage "https://github.com/seanseannery/opsfile"
  url "https://github.com/seanseannery/opsfile/archive/refs/tags/v0.8.1.tar.gz"
  sha256 "ba15ebb200b8c9f82839c90c204ee2bf82bb1fdc1840335b10a8edc4f557eaab"
  license "MIT"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X sean_seannery/opsfile/internal.Version=#{version}
      -X sean_seannery/opsfile/internal.Commit=none
    ]
    system "go", "build", *std_go_args(output: bin/"ops", ldflags: ldflags), "./cmd/ops"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/ops --version 2>&1")
  end
end
