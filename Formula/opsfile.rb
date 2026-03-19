class Opsfile < Formula
  desc "Like make/Makefile but for live operations commands"
  homepage "https://github.com/seanseannery/opsfile"
  url "https://github.com/seanseannery/opsfile/archive/refs/tags/v0.9.2.tar.gz"
  sha256 "aa28a520ddb995aedd85ebc20494492093d1056bf26c2bb83b81579e1a59a752"
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
