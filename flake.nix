{
  description = "Forte - A modern music player";

  # Pin to the same channel as NixOS 25.11 / Home Manager so go, mpv, GTK, and
  # WebKit reuse cache.nixos.org binaries instead of a second unstable tree.
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
    nixpkgs-go.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    pre-commit-hooks = {
      url = "github:cachix/git-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      nixpkgs-go,
      flake-utils,
      pre-commit-hooks,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        goPkgs = import nixpkgs-go { inherit system; };
        lib = pkgs.lib;
        go_1_25_11 = goPkgs.go_1_25;

        pre-commit = pre-commit-hooks.lib.${system}.run {
          src = ./.;
          excludes = [
            "(^|.*/)\\.go(/|$)"
            "(^|.*/)frontend/node_modules(/|$)"
            "(^|.*/)frontend/dist(/|$)"
            "(^|.*/)vendor(/|$)"
            "(^|.*/)frontend/test-results(/|$)"
          ];
          hooks = {
            # Fast, auto-fixing hygiene (pre-commit)
            trim-trailing-whitespace.enable = true;
            end-of-file-fixer.enable = true;
            check-merge-conflicts.enable = true;
            detect-private-keys.enable = true;
            check-added-large-files.enable = true;
            check-yaml.enable = true;
            check-shebang-scripts-are-executable.enable = true;

            gofmt.enable = true;

            nixfmt.enable = true;

            shellcheck = {
              enable = true;
              excludes = [ "\\.envrc$" ];
            };
            shfmt.enable = true;

            actionlint.enable = true;

            # Heavier checks aligned with CI (pre-push)
            golangci-lint-nocgo = {
              enable = true;
              name = "golangci-lint";
              entry = "${pkgs.golangci-lint}/bin/golangci-lint run --build-tags nocgo ./...";
              language = "system";
              pass_filenames = false;
              stages = [ "pre-push" ];
            };

            gotest = {
              enable = true;
              stages = [ "pre-push" ];
              settings.flags = "-tags nocgo";
            };

            svelte-check = {
              enable = true;
              name = "svelte-check";
              entry = ''
                bash -c '
                  if [ ! -d frontend/node_modules ]; then
                    echo "svelte-check: run cd frontend && npm ci first"
                    exit 1
                  fi
                  cd frontend && npm run check
                '
              '';
              files = "^frontend/.*\\.(svelte|ts)$";
              language = "system";
              pass_filenames = false;
              stages = [ "pre-push" ];
            };

            # Catch stale shas (vendorHash, npmDepsHash) before they are committed.
            # If go.mod / package-lock.json change, the build will fail with the new "got" hash
            # unless the pinned hash in flake.nix is updated.
            nix-build-forte = {
              enable = true;
              name = "nix build .#forte .#frontend (stale sha check for vendorHash / npmDepsHash)";
              entry = "nix build .#frontend .#forte --no-link";
              language = "system";
              pass_filenames = false;
              # Only when hashes might have changed — avoids a full Nix build on every push.
              files = "^(flake\\.nix|go\\.mod|go\\.sum|frontend/package-lock\\.json)$";
              stages = [ "pre-push" ];
            };
          };
        };

        frontend = pkgs.buildNpmPackage {
          pname = "forte-frontend";
          version = "1.0.0";
          src = ./frontend;
          nodejs = pkgs.nodejs_22;
          npmDepsHash = "sha256-MJDctjX8RZl72Vh3L0uPexNjuVHzLfZBMFJE+KzV1Lk=";
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
          version = "1.0.0";
          src = ./.;
          go = go_1_25_11;
          # Linux and Darwin vendoring differ: Linux applies Wails GTK patches in modBuildPhase.
          vendorHash =
            if pkgs.stdenv.isDarwin then
              "sha256-5ZcYXLLMFMb2DSiz9t4ghes8uFUQmH5Cw+tiSMRh5E8="
            else
              "sha256-ljBLN9G7yOFCdCd8AjAC6ZpO/ztSVxpCL/arQOSMor8=";
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
            ${lib.optionalString pkgs.stdenv.isLinux ''
              patch -p1 -d vendor/github.com/wailsapp/wails/v3 < ${./patches/wails-status-notifier-icon-name.patch}
              patch -p1 -d vendor/github.com/wailsapp/wails/v3 < ${./patches/wails-gtk4-transparent-window.patch}
            ''}

            mkdir -p vendor
            runHook postBuild
          '';
          tags = [
            "production"
            "nocgo"
          ]
          ++ lib.optionals pkgs.stdenv.isLinux [ "gtk4" ];
          ldflags = [
            "-s"
            "-w"
          ];
          subPackages = [ "." ];
          doCheck = false; # Tests need libmpv.so at runtime

          nativeBuildInputs =
            with pkgs;
            [
              pkg-config
              makeWrapper
            ]
            ++ lib.optionals pkgs.stdenv.isLinux [
              wrapGAppsHook4
              imagemagick
              desktop-file-utils
            ]
            ++ lib.optionals pkgs.stdenv.isDarwin [
              imagemagick
            ];

          buildInputs =
            with pkgs;
            [
              mpv
            ]
            ++ lib.optionals pkgs.stdenv.isLinux [
              gtk4
              webkitgtk_6_0
            ]
            ++ lib.optionals pkgs.stdenv.isDarwin [
              pkgs.apple-sdk_14
            ];

          preBuild = ''
            rm -rf frontend/dist
            mkdir -p frontend/dist
            cp -r ${frontend}/* frontend/dist/
          '';

          postInstall = ''
            ${lib.optionalString pkgs.stdenv.isLinux ''
              for size in 16 24 32 48 64 128 256 512; do
                install -d "$out/share/icons/hicolor/''${size}x''${size}/apps"
                magick build/appicon.png -resize "''${size}x''${size}" \
                  "$out/share/icons/hicolor/''${size}x''${size}/apps/io.github.willfish.forte.png"
                cp "$out/share/icons/hicolor/''${size}x''${size}/apps/io.github.willfish.forte.png" \
                  "$out/share/icons/hicolor/''${size}x''${size}/apps/forte.png"
              done
              install -Dm644 build/logo.svg $out/share/icons/hicolor/scalable/apps/io.github.willfish.forte.svg
              install -Dm644 build/logo.svg $out/share/icons/hicolor/scalable/apps/forte.svg
              install -Dm644 build/tray-idle.svg $out/share/icons/hicolor/scalable/apps/io.github.willfish.forte-tray-idle.svg
              install -Dm644 build/tray-playing.svg $out/share/icons/hicolor/scalable/apps/io.github.willfish.forte-tray-playing.svg
              for size in 16 24 32 48; do
                install -Dm644 "build/tray-idle-''${size}.png" \
                  "$out/share/icons/hicolor/''${size}x''${size}/apps/io.github.willfish.forte-tray-idle.png"
                install -Dm644 "build/tray-playing-''${size}.png" \
                  "$out/share/icons/hicolor/''${size}x''${size}/apps/io.github.willfish.forte-tray-playing.png"
              done
              for theme in green-dark green-light blue-dark blue-light financial-times-dark financial-times-light; do
                install -Dm644 "build/tray-''${theme}-idle-32.png" \
                  "$out/share/icons/hicolor/32x32/apps/io.github.willfish.forte-tray-''${theme}-idle.png"
                install -Dm644 "build/tray-''${theme}-playing-32.png" \
                  "$out/share/icons/hicolor/32x32/apps/io.github.willfish.forte-tray-''${theme}-playing.png"
              done
              install -Dm644 build/appicon.png $out/share/pixmaps/forte.png
              install -Dm644 build/linux/forte.desktop $out/share/applications/io.github.willfish.forte.desktop
              desktop-file-validate $out/share/applications/io.github.willfish.forte.desktop
            ''}

            ${lib.optionalString pkgs.stdenv.isDarwin ''
                            # Assemble macOS .app bundle (Approach 1) so the app has proper Dock icon
                            # from install time and does not require external wrappers for libmpv.
                            appDir="$out/Applications/Forte.app"
                            mkdir -p "$appDir/Contents"/{MacOS,Resources}

                            # Icon resources (png for association; iconset prepared for iconutil on mac builds if desired).
                            cp build/appicon.png "$appDir/Contents/Resources/appicon.png"
                            mkdir -p "$appDir/Contents/Resources/icon.iconset"
                            for s in 16 32 128 256 512; do
                              magick build/appicon.png -resize "''${s}x''${s}" -background none \
                                "$appDir/Contents/Resources/icon.iconset/icon_''${s}x''${s}.png"
                              magick build/appicon.png -resize "''${s}x''${s}" -background none \
                                "$appDir/Contents/Resources/icon.iconset/icon_''${s}x''${s}@2x.png"
                            done

                            # Info.plist for bundle identity and icon.
                            cat > "$appDir/Contents/Info.plist" << 'PLIST'
              <?xml version="1.0" encoding="UTF-8"?>
              <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
              <plist version="1.0">
              <dict>
              	<key>CFBundleDevelopmentRegion</key>
              	<string>en</string>
              	<key>CFBundleExecutable</key>
              	<string>forte</string>
              	<key>CFBundleIconFile</key>
              	<string>appicon</string>
              	<key>CFBundleIdentifier</key>
              	<string>io.github.willfish.forte</string>
              	<key>CFBundleInfoDictionaryVersion</key>
              	<string>6.0</string>
              	<key>CFBundleName</key>
              	<string>Forte</string>
              	<key>CFBundlePackageType</key>
              	<string>APPL</string>
              	<key>CFBundleShortVersionString</key>
                <string>1.0.0</string>
              	<key>CFBundleVersion</key>
                <string>1.0.0</string>
              	<key>LSMinimumSystemVersion</key>
              	<string>10.13</string>
              	<key>NSHighResolutionCapable</key>
              	<true/>
              </dict>
              </plist>
              PLIST

                            # Real binary + launcher wrapper that injects DYLD path for libmpv (meets AC: no consumer wrapper needed).
                            cp $out/bin/forte "$appDir/Contents/MacOS/forte-bin"
                            chmod +x "$appDir/Contents/MacOS/forte-bin"

                            makeWrapper "$appDir/Contents/MacOS/forte-bin" "$appDir/Contents/MacOS/forte" \
                              --prefix DYLD_LIBRARY_PATH : "${pkgs.lib.makeLibraryPath [ pkgs.mpv ]}"

                            # Keep a convenient bin/forte that is also wrapped.
                            rm -f $out/bin/forte || true
                            makeWrapper "$appDir/Contents/MacOS/forte" "$out/bin/forte" \
                              --prefix DYLD_LIBRARY_PATH : "${pkgs.lib.makeLibraryPath [ pkgs.mpv ]}"
            ''}
          '';

          # preFixup kept for Linux only (gapps); darwin wrapper done in postInstall.
          preFixup = lib.optionalString pkgs.stdenv.isLinux ''
            gappsWrapperArgs+=(
              --prefix LD_LIBRARY_PATH : "${pkgs.lib.makeLibraryPath [ pkgs.mpv ]}"
            )
          '';

          meta = with pkgs.lib; {
            description = "A modern desktop music player with local library and streaming server support";
            homepage = "https://github.com/willfish/forte";
            license = licenses.gpl3Only;
            maintainers = [ ];
            platforms = platforms.linux ++ platforms.darwin;
            mainProgram = "forte";
          };
        };

        vmUser = "forte";

        mkDesktopVmModule =
          desktop:
          { pkgs, ... }:
          {
            networking.hostName = "forte-${desktop}";
            system.stateVersion = "26.05";

            virtualisation.vmVariant.virtualisation = {
              cores = 2;
              memorySize = 4096;
              diskSize = 8192;
            };

            users.users.${vmUser} = {
              isNormalUser = true;
              description = "Forte VM validation user";
              extraGroups = [
                "audio"
                "video"
                "wheel"
              ];
              password = "";
            };

            security.polkit.enable = true;
            services.dbus.enable = true;

            services.pipewire = {
              enable = true;
              alsa.enable = true;
              pulse.enable = true;
            };

            hardware.graphics.enable = true;
            xdg.portal.enable = true;

            services.xserver.enable = true;
            services.displayManager.autoLogin = {
              enable = true;
              user = vmUser;
            };

            services.desktopManager.gnome.enable = desktop == "gnome";
            services.displayManager.gdm = lib.mkIf (desktop == "gnome") {
              enable = true;
            };

            services.desktopManager.plasma6.enable = desktop == "plasma";
            services.displayManager.sddm = lib.mkIf (desktop == "plasma") {
              enable = true;
              wayland.enable = false;
            };

            environment.systemPackages = with pkgs; [
              forte
              dbus
              desktop-file-utils
              hicolor-icon-theme
              playerctl
              xdg-utils
            ];
          };

        mkDesktopVm =
          desktop:
          nixpkgs.lib.nixosSystem {
            inherit system;
            modules = [ (mkDesktopVmModule desktop) ];
          };

        mkDesktopVmApp =
          desktop:
          let
            vm = mkDesktopVm desktop;
            runner = pkgs.writeShellApplication {
              name = "forte-vm-${desktop}";
              text = ''
                export QEMU_OPTS="''${QEMU_OPTS:-} -m 4096"
                exec ${vm.config.system.build.vm}/bin/run-*-vm "$@"
              '';
            };
          in
          {
            type = "app";
            program = "${runner}/bin/forte-vm-${desktop}";
            meta.description = "Launch Forte in an interactive ${desktop} validation VM";
          };

        mkDesktopVmSmoke =
          desktop:
          pkgs.testers.nixosTest {
            name = "forte-${desktop}-desktop-smoke";

            nodes.machine =
              { ... }:
              {
                imports = [ (mkDesktopVmModule desktop) ];

                virtualisation = {
                  cores = 2;
                  memorySize = 4096;
                  diskSize = 8192;
                };
              };

            testScript = ''
              machine.start()
              machine.wait_for_unit("display-manager.service")
              machine.wait_for_file("/run/user/1000/bus")
              machine.wait_for_x()

              machine.succeed("test -x /run/current-system/sw/bin/forte")
              machine.succeed("desktop-file-validate /run/current-system/sw/share/applications/io.github.willfish.forte.desktop")
              machine.succeed("test -e /run/current-system/sw/share/icons/hicolor/scalable/apps/io.github.willfish.forte.svg")
              machine.succeed("test -e /run/current-system/sw/share/icons/hicolor/scalable/apps/io.github.willfish.forte-tray-idle.svg")
              machine.succeed("test -e /run/current-system/sw/share/icons/hicolor/32x32/apps/io.github.willfish.forte-tray-financial-times-light-playing.png")

              base_env = "XDG_RUNTIME_DIR=/run/user/1000 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus"
              machine.succeed(
                  f"runuser -u ${vmUser} -- sh -lc '{base_env} systemctl --user show-environment > /tmp/forte-session.env'"
              )
              machine.succeed("grep -Eq '^(DISPLAY|WAYLAND_DISPLAY)=' /tmp/forte-session.env")

              env = "set -a; . /tmp/forte-session.env; set +a; export XDG_RUNTIME_DIR=/run/user/1000 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus"
              machine.succeed(
                  f"runuser -u ${vmUser} -- sh -lc '{env}; nohup forte >/tmp/forte.log 2>&1 &'"
              )
              machine.succeed(
                  "for i in $(seq 1 30); do pgrep -u ${vmUser} -af '[f]orte' && exit 0; sleep 1; done; cat /tmp/forte.log >&2 || true; exit 1"
              )
              machine.succeed(
                  f"for i in $(seq 1 30); do runuser -u ${vmUser} -- sh -lc '{env}; busctl --user --list | grep -q org.mpris.MediaPlayer2.forte' && exit 0; sleep 1; done; cat /tmp/forte.log >&2 || true; exit 1"
              )
              machine.succeed(
                  f"runuser -u ${vmUser} -- sh -lc '{env}; for i in $(seq 1 30); do svc=$(busctl --user --list | grep -Eo \"org.kde.StatusNotifierItem-[0-9]+-1\" | head -n1); test -n \"$svc\" && break; sleep 1; done; test -n \"$svc\"; busctl --user call -- \"$svc\" /StatusNotifierMenu com.canonical.dbusmenu GetLayout iias 0 -1 0 > /tmp/forte-tray-menu; grep -q \"Play/Pause\" /tmp/forte-tray-menu; grep -q \"Stop\" /tmp/forte-tray-menu; grep -q \"Next\" /tmp/forte-tray-menu; grep -q \"Previous\" /tmp/forte-tray-menu; grep -q \"Show/Hide Window\" /tmp/forte-tray-menu; grep -q \"Quit\" /tmp/forte-tray-menu'"
              )
              machine.screenshot("forte-${desktop}-launched")
              machine.succeed(
                  f"runuser -u ${vmUser} -- sh -lc '{env}; playerctl -p forte status >/tmp/forte-playerctl-status'"
              )
              machine.succeed("pkill -TERM -u ${vmUser} -f '[f]orte' || true")
            '';
          };

        devShellBase = {
          buildInputs = [
            go_1_25_11
          ]
          ++ pre-commit.enabledPackages
          ++ (with pkgs; [
            nodejs_22
            go-task
            golangci-lint
            git-cliff
            govulncheck
            ffmpeg
            pkg-config
            mpv
          ]);

          shellHook =
            pre-commit.shellHook
            + ''
              export GOPATH="$PWD/.go"
              export PATH="$GOPATH/bin:$PATH"
            ''
            + lib.optionalString pkgs.stdenv.isLinux ''
              export LD_LIBRARY_PATH="${pkgs.lib.makeLibraryPath [ pkgs.mpv ]}:$LD_LIBRARY_PATH"
            '';
        };

        devShellFull = devShellBase // {
          buildInputs =
            devShellBase.buildInputs
            ++ (
              with pkgs;
              lib.optionals pkgs.stdenv.isLinux [
                playwright-driver
                gtk3
                webkitgtk_4_1
                gtk4
                webkitgtk_6_0
              ]
              ++ lib.optionals pkgs.stdenv.isDarwin [
                apple-sdk_14
              ]
            );

          shellHook =
            devShellBase.shellHook
            + lib.optionalString pkgs.stdenv.isLinux ''
              export PLAYWRIGHT_BROWSERS_PATH="${pkgs.playwright-driver.browsers}"
              export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
              export CHROME_PATH="$(find -L "$PLAYWRIGHT_BROWSERS_PATH" -path '*/chrome-linux*/chrome' -type f | head -n 1)"
            '';
        };
      in
      {
        packages = {
          default = forte;
          forte = forte;
          frontend = frontend;
        };

        apps = lib.optionalAttrs pkgs.stdenv.isLinux {
          forte-vm-gnome = mkDesktopVmApp "gnome";
          forte-vm-plasma = mkDesktopVmApp "plasma";
        };

        checks = {
          pre-commit = pre-commit;
        }
        // lib.optionalAttrs pkgs.stdenv.isLinux {
          forte-vm-smoke-gnome = mkDesktopVmSmoke "gnome";
          forte-vm-smoke-plasma = mkDesktopVmSmoke "plasma";
        };

        formatter = pkgs.writeShellScriptBin "pre-commit-run" ''
          exec ${pre-commit.config.package}/bin/pre-commit run --all-files --config ${pre-commit.config.configFile}
        '';

        devShells = {
          # Fast path: Go, mpv, Node, linters — matches CI apt + task build workflow.
          default = pkgs.mkShell devShellBase;
          # Wails native link + Playwright e2e (pulls GTK/WebKit; use only when needed).
          full = pkgs.mkShell devShellFull;
        };
      }
    );
}
