{
  description = "Forte - A modern music player";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        frontend = pkgs.buildNpmPackage {
          pname = "forte-frontend";
          version = "0.1.0";
          src = ./frontend;
          npmDepsHash = "sha256-izThZlAW9yVoqlLB4qvs/G1epJqiMTESxQpVUfT19f8=";
          buildPhase = ''
            npm run build
          '';
          installPhase = ''
            mkdir -p $out
            cp -r dist/* $out/
          '';
        };

        forte = pkgs.buildGoModule {
          pname = "forte";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-9jLpRJQGgsUotSOS5ikr3WOvuXAqZWqmXRvI3ewn36c=";
          tags = [ "production" "nocgo" "gtk4" ];
          ldflags = [ "-s" "-w" ];
          subPackages = [ "." ];
          doCheck = false; # Tests need libmpv.so at runtime

          nativeBuildInputs = with pkgs; [
            pkg-config
            wrapGAppsHook4
          ];

          buildInputs = with pkgs; [
            gtk4
            webkitgtk_6_0
            mpv
          ];

          preBuild = ''
            rm -rf frontend/dist
            mkdir -p frontend/dist
            cp -r ${frontend}/* frontend/dist/
          '';

          postInstall = ''
            install -Dm644 build/appicon.png $out/share/icons/hicolor/1024x1024/apps/forte.png
            install -Dm644 build/linux/forte.desktop $out/share/applications/forte.desktop
          '';

          preFixup = ''
            gappsWrapperArgs+=(
              --prefix LD_LIBRARY_PATH : "${pkgs.lib.makeLibraryPath [ pkgs.mpv ]}"
            )
          '';

          meta = with pkgs.lib; {
            description = "A modern desktop music player with local library and streaming server support";
            homepage = "https://github.com/willfish/forte";
            license = licenses.gpl3Only;
            maintainers = [ ];
            platforms = platforms.linux;
            mainProgram = "forte";
          };
        };
      in
      {
        packages = {
          default = forte;
          forte = forte;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            nodejs_22
            go-task
            golangci-lint
            govulncheck
            pkg-config
            gtk4
            webkitgtk_6_0
            mpv
          ];

          shellHook = ''
            export GOPATH="$PWD/.go"
            export PATH="$GOPATH/bin:$PATH"
            export LD_LIBRARY_PATH="${pkgs.lib.makeLibraryPath [ pkgs.mpv ]}:$LD_LIBRARY_PATH"
          '';
        };
      });
}
