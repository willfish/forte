#!/usr/bin/env bash
set -euo pipefail

artifact_dir="${ARTIFACT_DIR:-/tmp/forte-distro-proof}"
host_uid="$(id -u)"
host_gid="$(id -g)"

usage() {
  cat <<'USAGE'
Usage: scripts/prove-distro-install.sh [ubuntu|arch|all]

Builds Forte inside fresh distro containers, installs the resulting package
inside fresh containers, validates desktop integration, and proves the app can
launch under Xvfb without relying on Nix-linked host artifacts.
USAGE
}

prepare_source() {
  local source_dir
  source_dir="$(mktemp -d)"

  git ls-files -z | tar --null -T - -cf - | tar -x -C "$source_dir"
  printf '%s\n' "$source_dir"
}

cleanup_source() {
  local source_dir="$1"

  docker run --rm -v "$source_dir:/src" ubuntu:latest \
    bash -lc 'rm -rf /src/* /src/.[!.]* /src/..?* 2>/dev/null || true' >/dev/null
  rmdir "$source_dir"
}

build_ubuntu() {
  local source_dir="$1"

  docker run --rm -i \
    -e HOST_UID="$host_uid" \
    -e HOST_GID="$host_gid" \
    -v "$source_dir:/src" \
    -v "$artifact_dir:/artifacts" \
    -w /src \
    ubuntu:latest \
    bash -s <<'EOF'
set -euxo pipefail

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y \
  build-essential \
  ca-certificates \
  git \
  golang-go \
  libgtk-4-dev \
  libmpv-dev \
  libwebkitgtk-6.0-dev \
  nodejs \
  npm \
  pkg-config

export PATH="$HOME/go/bin:$PATH"
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.95
go install github.com/go-task/task/v3/cmd/task@v3.51.1

export VERSION=0.1.0-test
export GIT_COMMITTER_NAME=Forte
export GIT_COMMITTER_EMAIL=forte@example.invalid
task linux:create:deb

ldd bin/forte | tee /tmp/forte-ldd.txt
! grep -q /nix/store /tmp/forte-ldd.txt
test -f bin/forte.deb

install -m644 bin/forte.deb /artifacts/forte-ubuntu.deb
chown "$HOST_UID:$HOST_GID" /artifacts/forte-ubuntu.deb
chown -R "$HOST_UID:$HOST_GID" /src
EOF
}

prove_ubuntu() {
  docker run --rm --privileged -i \
    -v "$artifact_dir:/artifacts:ro" \
    ubuntu:latest \
    bash -s <<'EOF'
set -euxo pipefail

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y /artifacts/forte-ubuntu.deb desktop-file-utils xvfb dbus-x11 procps

command -v forte
desktop-file-validate /usr/share/applications/io.github.willfish.forte.desktop
test -e /usr/share/icons/hicolor/scalable/apps/io.github.willfish.forte.svg
test -e /usr/share/icons/hicolor/scalable/apps/io.github.willfish.forte-tray-idle.svg
ldd /usr/local/bin/forte | tee /tmp/forte-ldd.txt >/dev/null
! grep -q /nix/store /tmp/forte-ldd.txt

set +e
timeout 15s xvfb-run -a dbus-run-session env WEBKIT_DISABLE_DMABUF_RENDERER=1 forte >/tmp/forte.log 2>&1
status=$?
set -e
cat /tmp/forte.log
test "$status" -eq 124
EOF
}

build_arch() {
  local source_dir="$1"

  docker run --rm -i \
    -e HOST_UID="$host_uid" \
    -e HOST_GID="$host_gid" \
    -v "$source_dir:/src" \
    -v "$artifact_dir:/artifacts" \
    -w /src \
    archlinux:latest \
    bash -s <<'EOF'
set -euxo pipefail

pacman -Syu --noconfirm
pacman -S --noconfirm --needed \
  base-devel \
  git \
  go \
  gtk4 \
  mpv \
  nodejs \
  npm \
  pkgconf \
  webkitgtk-6.0

export PATH="$HOME/go/bin:$PATH"
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.95
go install github.com/go-task/task/v3/cmd/task@v3.51.1

export VERSION=0.1.0-test
export GIT_COMMITTER_NAME=Forte
export GIT_COMMITTER_EMAIL=forte@example.invalid
task linux:create:aur

ldd bin/forte | tee /tmp/forte-ldd.txt
! grep -q /nix/store /tmp/forte-ldd.txt
test -f bin/forte.pkg.tar.zst

install -m644 bin/forte.pkg.tar.zst /artifacts/forte-arch.pkg.tar.zst
chown "$HOST_UID:$HOST_GID" /artifacts/forte-arch.pkg.tar.zst
chown -R "$HOST_UID:$HOST_GID" /src
EOF
}

prove_arch() {
  docker run --rm --privileged -i \
    -v "$artifact_dir:/artifacts:ro" \
    archlinux:latest \
    bash -s <<'EOF'
set -euxo pipefail

pacman -Syu --noconfirm
pacman -U --noconfirm /artifacts/forte-arch.pkg.tar.zst
pacman -S --noconfirm --needed desktop-file-utils xorg-server-xvfb xorg-xauth dbus procps-ng

command -v forte
desktop-file-validate /usr/share/applications/io.github.willfish.forte.desktop
test -e /usr/share/icons/hicolor/scalable/apps/io.github.willfish.forte.svg
test -e /usr/share/icons/hicolor/scalable/apps/io.github.willfish.forte-tray-idle.svg
ldd /usr/local/bin/forte | tee /tmp/forte-ldd.txt >/dev/null
! grep -q /nix/store /tmp/forte-ldd.txt

set +e
timeout 15s xvfb-run -a dbus-run-session env WEBKIT_DISABLE_DMABUF_RENDERER=1 forte >/tmp/forte.log 2>&1
status=$?
set -e
cat /tmp/forte.log
test "$status" -eq 124
EOF
}

prove_distro() {
  local distro="$1"
  local source_dir

  source_dir="$(prepare_source)"
  trap 'cleanup_source "$source_dir"' RETURN
  mkdir -p "$artifact_dir"

  case "$distro" in
    ubuntu)
      build_ubuntu "$source_dir"
      prove_ubuntu
      ;;
    arch)
      build_arch "$source_dir"
      prove_arch
      ;;
    *)
      printf 'Unknown distro: %s\n' "$distro" >&2
      usage >&2
      exit 2
      ;;
  esac

  trap - RETURN
  cleanup_source "$source_dir"
}

target="${1:-all}"

case "$target" in
  ubuntu|arch)
    prove_distro "$target"
    ;;
  all)
    prove_distro ubuntu
    prove_distro arch
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    printf 'Unknown target: %s\n' "$target" >&2
    usage >&2
    exit 2
    ;;
esac
