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
        lib = pkgs.lib;

        frontend = pkgs.buildNpmPackage {
          pname = "forte-frontend";
          version = "0.1.0";
          src = ./frontend;
          nodejs = pkgs.nodejs_22;
          npmDepsHash = "sha256-U5x+/CNX2I9FqIasFdiRzhg2NqcnHljBBUPfUaLDyi8=";
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
          vendorHash = "sha256-SX43UbVi1YEC323j/rvE6OgjA8G/RfaXoNACVhL7B44=";
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
            patch -p1 -d vendor/github.com/wailsapp/wails/v3 < ${./patches/wails-status-notifier-icon-name.patch}

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
            imagemagick
            desktop-file-utils
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
            install -Dm644 build/appicon.png $out/share/pixmaps/forte.png
            install -Dm644 build/linux/forte.desktop $out/share/applications/io.github.willfish.forte.desktop
            desktop-file-validate $out/share/applications/io.github.willfish.forte.desktop
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
            extraGroups = [ "audio" "video" "wheel" ];
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

        mkDesktopVm = desktop:
          nixpkgs.lib.nixosSystem {
            inherit system;
            modules = [ (mkDesktopVmModule desktop) ];
          };

        mkDesktopVmApp = desktop:
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

        mkDesktopVmSmoke = desktop:
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

        checks = lib.optionalAttrs pkgs.stdenv.isLinux {
          forte-vm-smoke-gnome = mkDesktopVmSmoke "gnome";
          forte-vm-smoke-plasma = mkDesktopVmSmoke "plasma";
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
