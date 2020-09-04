class Gotk < Formula
  desc "Experimental toolkit for assembling CD pipelines the GitOps way"
  homepage "https://toolkit.fluxcd.io/"
  url "https://github.com/fluxcd/toolkit.git",
      tag: "0.0.24",
      revision: "d6c6c88e6e08d0ab7fa89c69241f1b1ca5aa658b"
  head "https://github.com/fluxcd/toolkit.git"
  license "Apache-2.0"

  depends_on "go" => :build
  depends_on "kubernetes-cli"

  bottle :unneeded

  def install
    if build.stable?
      bin.install "gotk"
    else
      ENV["CGO_ENABLED"] = "0"
      cd buildpath/"cmd/gotk" do
        system "go", "build", "-ldflags", "-s -w -X main.VERSION=#{version}", "-trimpath", "-o", bin/"gotk"
      end
    end
  end

  on_linux do
    stable do
      url "https://github.com/fluxcd/toolkit/releases/download/v0.0.24/gotk_0.0.24_linux_amd64.tar.gz"
      sha256 "05684cbb7fb51c4695e52006a06fd8abdece1c41baba27d0867a4d3416491bb2"
    end
  end

  on_macos do
    stable do
      url "https://github.com/fluxcd/toolkit/releases/download/v0.0.24/gotk_0.0.24_darwin_amd64.tar.gz"
      sha256 "fcd6e68062bf811da031020ea7bb8640986b6407b66aa755af45d913e18c075e"
    end
  end

  test do
    run_output = shell_output("#{bin}/gotk 2>&1")
    assert_match "Command line utility for assembling Kubernetes CD pipelines the GitOps way.", run_output

    version_output = shell_output("#{bin}/gotk --version 2>&1")
    assert_match version.to_s, version_output

    system "#{bin}/gotk", "install", "--export"
  end
end
