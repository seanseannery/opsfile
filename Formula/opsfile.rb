class Opsfile < Formula
  desc "Like make/Makefile but for live operations commands"
  homepage "https://github.com/seanseannery/opsfile"
  url "https://github.com/seanseannery/opsfile/archive/refs/tags/v0.8.5.tar.gz"
  sha256 "0a8587c4895fa9d1082a0beb9b52de4005f72bbc88fb3ea923aacc75c48109e2"
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
