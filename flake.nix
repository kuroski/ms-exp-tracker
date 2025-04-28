
  description = "Wails 3 App with bundled Tesseract";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            pkgs.nodejs
            pkgs.tesseract
            pkgs.appdmg
          ];

          shellHook = ''
            export CGO_ENABLED=1
            export GO111MODULE=on
            export PATH=$PATH:$(go env GOPATH)/bin

            # Install Wails v3 if not installed
            if ! [ -f "$(go env GOPATH)/bin/wails3" ]; then
              echo "Installing Wails v3 CLI..."
              go install github.com/wailsapp/wails/v3/cmd/wails3@latest
            fi

            # Optionally symlink wails3 -> wails
            if ! [ -f "$(go env GOPATH)/bin/wails" ]; then
              ln -s $(go env GOPATH)/bin/wails3 $(go env GOPATH)/bin/wails
            fi
          '';
        };
      }
    );
}
