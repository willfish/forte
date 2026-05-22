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
          nodejs = pkgs.nodejs_22;
          npmDepsHash = "sha256-7w3PrVtvOrmu5TWnUoSOMvZOs8bsTLeOEztwED9wc6k=";
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
          go = pkgs.go_1_25;
          vendorHash = "sha256-7w3PrVtvOrmu5TWnUoSOMvZOs8bsTLeOEztwED9wc6k=";
          modBuildPhase = ''
            runHook preBuild

            if [ -d vendor ]; then
              echo "vendor folder exists, please set 'vendorHash = null;' in your expression"
              exit 10
            fi

            export GIT_SSL_CAINFO=$NIX_SSL_CERT_FILE
            go mod download

            webview2Loader="$GOPATH/pkg/mod/github.com/wailsapp/wails/webview2@v1.0.24/webviewloader"
            chmod -R u+w "$GOPATH/pkg/mod/github.com/wailsapp/wails/webview2@v1.0.24"
            mkdir -p "$webview2Loader/x86" "$webview2Loader/x64" "$webview2Loader/arm64"
            : > "$webview2Loader/x86/WebView2Loader.dll"
            : > "$webview2Loader/x64/WebView2Loader.dll"
            : > "$webview2Loader/arm64/WebView2Loader.dll"

            if (( "''${NIX_DEBUG:-0}" >= 1 )); then
              goModVendorFlags+=(-v)
            fi
            go mod vendor "''${goModVendorFlags[@]}"

            mkdir -p vendor
            runHook postBuild
          '';
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
          frontend = frontend;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_25
            nodejs_22
            playwright-driver
            go-task
            golangci-lint
            govulncheck
            ffmpeg
            pkg-config
            gtk3
            webkitgtk_4_1
            gtk4
            webkitgtk_6_0
            mpv
          ];

          shellHook = ''
            export GOPATH="$PWD/.go"
            export PATH="$GOPATH/bin:$PATH"
            export LD_LIBRARY_PATH="${pkgs.lib.makeLibraryPath [ pkgs.mpv ]}:$LD_LIBRARY_PATH"
            export PLAYWRIGHT_BROWSERS_PATH="${pkgs.playwright-driver.browsers}"
            export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
            export CHROME_PATH="$(find -L "$PLAYWRIGHT_BROWSERS_PATH" -path '*/chrome-linux*/chrome' -type f | head -n 1)"
          '';
        };
      });
}
