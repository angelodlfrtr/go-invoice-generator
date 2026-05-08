{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    mustangproject.url = "github:/angelodlfrtr/mustangproject-nix-flake";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      mustangproject,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let

        pkgs = import nixpkgs {
          inherit system;

          overlays = [
            (final: prev: {
              mustang-cli = mustangproject.outputs.packages.${system}.default;
            })
          ];
        };
      in
      {
        formatter = pkgs.nixfmt-rfc-style;

        # Dev shell
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            golangci-lint
            golangci-lint-langserver
            gosec
            dprint
            gnumake
            verapdf
            mustang-cli
          ];
        };
      }
    );
}
