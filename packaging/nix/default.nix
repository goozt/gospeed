{ lib, buildGoModule, fetchFromGitHub }:

buildGoModule rec {
  pname = "gospeed";
  version = "1.3.2";

  src = fetchFromGitHub {
    owner = "goozt";
    repo = "gospeed";
    rev = "v${version}";
    hash = ""; # Replace with: nix-prefetch-url --unpack https://github.com/goozt/gospeed/archive/v${version}.tar.gz
  };

  vendorHash = ""; # Replace with: nix run nixpkgs#nix-prefetch -- '{ sha256 }: (import <nixpkgs> {}).buildGoModule { ... vendorHash = sha256; }'

  subPackages = [ "cmd/gospeed" "cmd/gospeed-server" ];

  ldflags = [
    "-s" "-w"
    "-X github.com/goozt/gospeed/internal/version.Version=${version}"
  ];

  meta = with lib; {
    description = "Fast, zero-dependency network speed testing tool";
    homepage = "https://github.com/goozt/gospeed";
    license = licenses.mit;
    maintainers = [ ];
    mainProgram = "gospeed";
  };
}
