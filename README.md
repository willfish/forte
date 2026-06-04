# Forte

Forte is a Linux desktop music player built with Go, Wails, Svelte, mpv, and SQLite. It plays internet radio out of the box, and can optionally manage a local music library with Subsonic and Jellyfin servers in the same collection.

<img width="1408" height="1910" alt="image" src="https://github.com/user-attachments/assets/0eaf6e07-4e4c-4ecd-b52b-318c80aca1fa" />

## What It Does

- **Radio first** - browse Radio Browser and SomaFM stations, filter by country or codec, save favourites, pin stations, add custom streams, and keep playback history.
- **Library mode when you want it** - scan local FLAC, MP3, Ogg, Opus, and WAV files, then browse albums, artists, tracks, playlists, and listening stats.
- **Streaming libraries** - connect Subsonic-compatible and Jellyfin servers, test credentials, sync catalogues, and play remote tracks alongside local files.
- **Playback built on mpv** - queue management, shuffle, repeat, seeking, volume control, ReplayGain support, and gapless-style queue preloading.
- **Desktop integration** - MPRIS controls, desktop notifications, a COSMIC-friendly themed tray icon, launcher icons, and configurable dark/light colour themes.
- **Scrobbling** - Last.fm and ListenBrainz support with now-playing updates and a retry queue for missed scrobbles.
- **Metadata extras** - full-text search, CUE sheet parsing, play history, top artists/albums/tracks, and cached artist information from Last.fm and MusicBrainz.

## Installation

### Linux Installer

Install Forte on Ubuntu, Debian, or Arch:

```sh
curl -fsSL https://raw.githubusercontent.com/willfish/forte/master/scripts/install.sh | sh
```

### Nix

Build and run the current release from GitHub:

```sh
nix run github:willfish/forte
```

Or build the package:

```sh
nix build github:willfish/forte
```

In a NixOS or Home Manager flake, add the package from the Forte flake:

```nix
inputs.forte.url = "github:willfish/forte";

environment.systemPackages = [
  inputs.forte.packages.${pkgs.system}.default
];
```

### From Source

Forte supports Linux and macOS (via Nix). The Nix flake provides a first-class `Forte.app` on darwin (proper Dock icon, menubar status item with menu on click, playback state, self-contained libmpv).

The flake pins **nixos-25.11** (same channel as current NixOS/Home Manager) so `go`, `mpv`, and friends come from **cache.nixos.org** instead of a second unstable nixpkgs tree.

```sh
# Fast dev shell: Go, Node, mpv, linters (no GTK/WebKit download)
nix develop

# Full shell when linking Wails or running Playwright in Nix
nix develop .#full

task build
./bin/forte
```

To avoid compiling `.#forte` locally (e.g. in Home Manager), use the optional Cachix cache after it is configured (see `.github/workflows/nix-cache.yml`):

```sh
cachix use willfish-forte   # once the cache exists
nix build .#forte
```

Without Nix, install:

- Go 1.25+
- Node.js 22+
- GTK4
- WebKitGTK 6.0
- mpv
- pkg-config
- [go-task](https://taskfile.dev)

The build tasks install the pinned Wails 3 CLI into the repo-local `.go/bin`
directory when needed.

Then build:

```sh
cd frontend
npm ci
cd ..
task build
```

## Development

```sh
# Hot reload with Wails and Vite
task dev

# Build the Linux desktop app
task build

# Seed demo data for screenshots and UI testing
task demo

# Go tests
go test -tags nocgo ./...

# Frontend type checking
cd frontend && npm run check

# Lint and vulnerability checks
golangci-lint run --build-tags nocgo
govulncheck -tags nocgo ./...
```

The demo task creates a fixture library with albums, tracks, playlists, play history, and radio data. It is safe to run more than once.

## Architecture

```
frontend/          Svelte 5 UI, Vite build, Wails bindings
internal/library/  SQLite schema, scans, search, playlists, stats, servers
internal/player/   mpv engine and queue logic
internal/radio/    Radio Browser and SomaFM clients
internal/system/   MPRIS and desktop notifications
internal/metadata/ Audio tag reading
internal/scrobbling/
                   Last.fm and ListenBrainz clients
internal/streaming/
                   Subsonic and Jellyfin clients
libraryservice.go  Library, radio, settings, and integration RPC methods
playerservice.go   Playback, queue, tray, MPRIS, radio, and scrobbling runtime
main.go            Wails application setup
```

The backend owns I/O, playback, database access, desktop integration, and external services. The frontend is a Svelte app that talks to the backend through generated Wails bindings. User data is stored in SQLite under the app configuration directory.

## Project Status

Forte is a personal Linux desktop app and is evolving quickly. Issues and feature requests are welcome at [github.com/willfish/forte/issues](https://github.com/willfish/forte/issues).

## License

[GNU General Public License v3.0](LICENSE)
