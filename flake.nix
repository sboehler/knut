{
  description = "knut";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs = { self, nixpkgs, ... }@inputs:
  let
    supportedSystems = ["x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin"];

    forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

    nixpkgsFor = forAllSystems (system: import nixpkgs {
      inherit system;
      config = { };
    });

  in {
    devShells = forAllSystems (system:
    let
      pkgs = nixpkgsFor.${system};
    in {
      default = pkgs.mkShell {
        name = "knut";
        buildInputs = with pkgs; [
          go
        ];
      };
    });
  };
}
