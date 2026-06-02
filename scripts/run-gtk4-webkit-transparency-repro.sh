#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

cc scripts/gtk4-webkit-transparency-repro.c \
  -o /tmp/forte-gtk4-webkit-transparency-repro \
  $(pkg-config --cflags --libs gtk4 webkitgtk-6.0)

exec /tmp/forte-gtk4-webkit-transparency-repro "$@"
