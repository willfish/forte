#!/usr/bin/env sh
set -eu

script_dir="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"

FORTE_INSTALLER_TESTING=1
export FORTE_INSTALLER_TESTING
# shellcheck disable=SC1091
. "$script_dir/install.sh"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

pass_count=0

pass() {
	pass_count=$((pass_count + 1))
}

assert_eq() {
	got="$1"
	want="$2"
	label="$3"

	if [ "$got" != "$want" ]; then
		printf 'not ok - %s: got %s, want %s\n' "$label" "$got" "$want" >&2
		exit 1
	fi
	pass
}

printf 'package\n' >"$tmpdir/forte.deb"
hash="$(sha256sum "$tmpdir/forte.deb" | awk '{ print $1 }')"
printf '%s  forte.deb\n%s  forte.pkg.tar.zst\n' "$hash" "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" >"$tmpdir/checksums-sha256.txt"

assert_eq "$(extract_checksum "$tmpdir/checksums-sha256.txt" forte.deb)" "$hash" "extracts package checksum"

verify_checksum "$tmpdir/forte.deb" "$hash"
pass

if (verify_checksum "$tmpdir/forte.deb" "0000000000000000000000000000000000000000000000000000000000000000") 2>"$tmpdir/mismatch.log"; then
	printf 'not ok - checksum mismatch should fail\n' >&2
	exit 1
fi
if ! grep -q "checksum mismatch" "$tmpdir/mismatch.log"; then
	printf 'not ok - checksum mismatch error was not reported\n' >&2
	exit 1
fi
pass

if extract_checksum "$tmpdir/checksums-sha256.txt" missing.deb >/dev/null 2>&1; then
	printf 'not ok - missing asset checksum should fail\n' >&2
	exit 1
fi
pass

printf 'ok - %s installer tests passed\n' "$pass_count"
