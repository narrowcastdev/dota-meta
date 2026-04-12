{
  description = "dota-meta — Dota 2 bracket-specific meta analyzer";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go
            pkgs.gopls
            pkgs.gotools
            pkgs.nodejs_20
            pkgs.yarn
          ];

          shellHook = ''
            echo "dota-meta dev shell — $(go version)"
          '';
        };
      }
    );
}
