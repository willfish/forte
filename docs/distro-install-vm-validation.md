# Distro Install VM Validation

Forte should have a weekly validation path that installs the app the same way a
non-Nix user would install it, then launches it in a real desktop session.

The existing NixOS VM checks prove the flake package works in GNOME and Plasma.
This validation is different: it should prove the public install instructions work
on current mainstream Linux desktops.

## Target Coverage

- Ubuntu latest LTS with GNOME.
- Arch Linux latest rolling image with GNOME.
- x86_64 initially.
- Weekly scheduled runs and manual dispatch, not every PR.

## Expected User Flow

The VM test should run the same command documented in the README, for example:

```sh
curl -fsSL https://raw.githubusercontent.com/willfish/forte/master/install.sh | sh
```

That installer does not exist yet, so the first implementation should land the
installer and then make the VM harness consume it verbatim. The VM must not install
the Nix package or reach into the repository build output as its primary path.

## Nix Shape

Nix can still own the harness:

- Provide a `nix run` app that launches a QEMU VM for a named distro.
- Use Nix-provided QEMU, cloud-localds, ssh, xvfb/screenshot tooling, and helper
  scripts so the runner environment is reproducible.
- Fetch official distro images in the workflow or through fixed-output Nix fetches.
- Generate cloud-init seed images for users, SSH keys, packages, and first-boot
  setup.
- Run the public install command inside the guest.
- Launch a GNOME session and start Forte from the installed desktop entry.
- Assert the installed binary, desktop file, icon, MPRIS name, and `playerctl`
  integration.
- Capture screenshots and logs as artifacts.

Nix should orchestrate Ubuntu and Arch VMs; it should not try to turn them into
NixOS systems. If we use "latest" images, the scheduled workflow should be treated
as moving-environment validation. If we want fully reproducible failures, pin the
image URL and hash and update that pin explicitly.

## Suggested Test Contract

Each distro job should produce:

- `/tmp/forte-install.log`
- `/tmp/forte-launch.log`
- `forte-<distro>-gnome-launched.png`
- the image/version metadata used for the run

Each distro job should verify:

- `forte` exists on `PATH`.
- `io.github.willfish.forte.desktop` is installed and validates where the distro
  provides `desktop-file-validate`.
- the application icon is visible through the hicolor icon theme.
- Forte launches inside the logged-in GNOME session.
- `org.mpris.MediaPlayer2.forte` appears on the user bus.
- `playerctl -p forte status` can talk to Forte.

## Open Work

- Add the public curl installer and README instructions.
- Decide whether the installer downloads release assets from the latest GitHub
  release or supports installing a CI-built package artifact.
- Publish release assets for Debian/Ubuntu and Arch package formats.
- Add the QEMU/cloud-init harness and scheduled workflow jobs.
