#!/usr/bin/env sh
set -eu

repo="${FORTE_REPO:-willfish/forte}"
version="${FORTE_VERSION:-latest}"
package_url="${FORTE_PACKAGE_URL:-}"
package_sha256="${FORTE_PACKAGE_SHA256:-}"
checksum_url="${FORTE_CHECKSUM_URL:-}"

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

extract_checksum() {
	checksums_file="$1"
	asset_name="$2"

	awk -v asset="$asset_name" '$2 == asset { print $1; found = 1; exit } END { if (!found) exit 1 }' "$checksums_file"
}

verify_checksum() {
	file="$1"
	expected="$2"

	if [ -z "$expected" ]; then
		die "missing checksum for downloaded package"
	fi
	if ! command -v sha256sum >/dev/null 2>&1; then
		die "sha256sum is required to verify Forte downloads"
	fi

	actual="$(sha256sum "$file" | awk '{ print $1 }')"
	if [ "$actual" != "$expected" ]; then
		die "checksum mismatch for $(basename "$file"): expected $expected, got $actual"
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
	*" debian "* | *" ubuntu "*)
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

main() {
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

	if [ -z "$package_sha256" ] && [ -z "${FORTE_PACKAGE_URL:-}" ]; then
		if [ -z "$checksum_url" ]; then
			checksum_url="$(release_asset_url "checksums-sha256.txt")"
		fi
		checksums="$tmpdir/checksums-sha256.txt"
		log "Downloading Forte checksums from $checksum_url"
		download "$checksum_url" "$checksums"
		package_sha256="$(extract_checksum "$checksums" "$asset")" || die "checksum for $asset not found in checksums-sha256.txt"
	fi

	log "Downloading Forte from $package_url"
	download "$package_url" "$package"

	if [ -n "$package_sha256" ]; then
		log "Verifying Forte package checksum"
		verify_checksum "$package" "$package_sha256"
	fi

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
}

if [ "${FORTE_INSTALLER_TESTING:-0}" != "1" ]; then
	main "$@"
fi
