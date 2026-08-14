{
  description = "Forte — desktop music player for radio and local or streaming libraries";

  # mpv/GTK/WebKit come from nixos-25.11 (cache.nixos.org). Go 1.26 is taken
  # from nixpkgs-go (nixos-unstable) until 25.11 carries that toolchain.
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
        # Track latest Go from nixpkgs-go (unstable). Rename when major bumps.
        goToolchain = goPkgs.go_1_26;

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

        forteSrc = lib.cleanSourceWith {
          src = ./.;
          filter =
            path: type:
            let
              rel = lib.removePrefix (toString ./. + "/") (toString path);
            in
            lib.cleanSourceFilter path type
            && !lib.hasPrefix "frontend/node_modules" rel
            && !lib.hasPrefix "frontend/test-results" rel
            && !lib.hasPrefix ".go" rel
            && !lib.hasPrefix "advisor-plans" rel
            && !lib.hasPrefix "cmd/stresstest" rel;
        };

        forte = pkgs.callPackage ./nix/package.nix {
          src = forteSrc;
          version = "1.1.0";
          buildGoModule = pkgs.buildGoModule.override { go = goToolchain; };
          go = goToolchain;
          npmDepsHash = "sha256-2AQVj9sZNYmOx9Qwln7cg7kzVErKg3nQzvVGqWuWPnA=";
          vendorHash =
            if pkgs.stdenv.isDarwin then
              # Updated on Darwin CI if this mismatches after go.mod changes.
              "sha256-1zMhKwEbh5ef9tjDumsX1bsFvrMk2QvaHyTFqDwVc6E="
            else
              "sha256-QlhQms3GnvLVvpQF4r2uEfBOMCCSMtq87PJzo/hmT2k=";
        };
        frontend = forte.frontend;

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
            goToolchain
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
