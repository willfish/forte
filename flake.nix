{
  description = "Forte - A modern music player";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
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
      flake-utils,
      pre-commit-hooks,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        lib = pkgs.lib;

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
          };
        };

        frontend = pkgs.buildNpmPackage {
          pname = "forte-frontend";
          version = "0.1.0";
          src = ./frontend;
          nodejs = pkgs.nodejs_22;
          npmDepsHash = "sha256-kEprDYBdn3cayhPsS81DOEs+biGUquJZJvO2jD07Lp0=";
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
          vendorHash =
            if pkgs.stdenv.isDarwin then
              "sha256-M356Dg/QaaeitWx7srKTH+Hpht+c/HfHLdSfwaGwJus="
            else
              "sha256-PgFI1Kw2jw47OARij8R0LXGthSa/LItw5rNdOJCwb3k=";
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
            ]
            ++ lib.optionals pkgs.stdenv.isLinux [
              wrapGAppsHook4
              imagemagick
              desktop-file-utils
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

          postInstall = lib.optionalString pkgs.stdenv.isLinux ''
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
          '';

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

        mkDesktopVmModule = desktop: { pkgs, ... }: {
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

            nodes.machine = { ... }: {
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
          exec ${pre-commit.package}/bin/pre-commit run --all-files --config ${pre-commit.configFile}
        '';

        devShells.default = pkgs.mkShell {
          buildInputs =
            pre-commit.enabledPackages
            ++ (
              with pkgs;
              [
                go_1_25
                nodejs_22
                go-task
                golangci-lint
                govulncheck
                ffmpeg
                pkg-config
                mpv
              ]
              ++ lib.optionals pkgs.stdenv.isLinux [
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
            pre-commit.shellHook
            + ''
              export GOPATH="$PWD/.go"
              export PATH="$GOPATH/bin:$PATH"
            ''
            + lib.optionalString pkgs.stdenv.isLinux ''
              export LD_LIBRARY_PATH="${pkgs.lib.makeLibraryPath [ pkgs.mpv ]}:$LD_LIBRARY_PATH"
              export PLAYWRIGHT_BROWSERS_PATH="${pkgs.playwright-driver.browsers}"
              export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
              export CHROME_PATH="$(find -L "$PLAYWRIGHT_BROWSERS_PATH" -path '*/chrome-linux*/chrome' -type f | head -n 1)"
            '';
        };
      }
    );
}
