#!/usr/bin/env sh
set -eu

repo="${FORTE_REPO:-willfish/forte}"
version="${FORTE_VERSION:-latest}"
package_url="${FORTE_PACKAGE_URL:-}"

log() {
  printf '%s\n' "$*"
}

die() {
  printf 'forte installer: %s\n' "$*" >&2
  exit 1
}

as_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  elif command -v sudo >/dev/null 2>&1; then
    sudo "$@"
  else
    die "this installer needs root privileges; rerun as root or install sudo"
  fi
}

download() {
  url="$1"
  output="$2"

  case "$url" in
    file://*)
      cp "${url#file://}" "$output"
      ;;
    *)
      if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$output"
      elif command -v wget >/dev/null 2>&1; then
        wget -qO "$output" "$url"
      else
        die "curl or wget is required to download Forte"
      fi
      ;;
  esac
}

release_asset_url() {
  asset="$1"

  if [ "$version" = "latest" ]; then
    printf 'https://github.com/%s/releases/latest/download/%s\n' "$repo" "$asset"
  else
    printf 'https://github.com/%s/releases/download/%s/%s\n' "$repo" "$version" "$asset"
  fi
}

install_debian() {
  package="$1"

  export DEBIAN_FRONTEND=noninteractive
  as_root apt-get update
  as_root apt-get install -y "$package"
}

install_arch() {
  package="$1"

  as_root pacman -Syu --noconfirm
  as_root pacman -U --noconfirm "$package"
}

detect_distro() {
  if [ -r /etc/os-release ]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    distro_id="${ID:-}"
    distro_like="${ID_LIKE:-}"
  else
    distro_id=""
    distro_like=""
  fi

  case " $distro_id $distro_like " in
    *" debian "*|*" ubuntu "*)
      printf 'debian\n'
      ;;
    *" arch "*)
      printf 'arch\n'
      ;;
    *)
      die "unsupported Linux distribution; expected Debian, Ubuntu, or Arch"
      ;;
  esac
}

distro="$(detect_distro)"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

case "$distro" in
  debian)
    asset="forte.deb"
    package="$tmpdir/$asset"
    ;;
  arch)
    asset="forte.pkg.tar.zst"
    package="$tmpdir/$asset"
    ;;
  *)
    die "unsupported distro: $distro"
    ;;
esac

if [ -z "$package_url" ]; then
  package_url="$(release_asset_url "$asset")"
fi

log "Downloading Forte from $package_url"
download "$package_url" "$package"

case "$distro" in
  debian)
    log "Installing Forte with apt"
    install_debian "$package"
    ;;
  arch)
    log "Installing Forte with pacman"
    install_arch "$package"
    ;;
esac

log "Forte installed successfully"
