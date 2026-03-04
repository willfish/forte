# Forte

A desktop music player for Linux built with Go and Wails. Plays local files, streams from Subsonic and Jellyfin servers, and scrobbles to Last.fm and ListenBrainz.

## Features

- **Local library** - scan directories for FLAC, MP3, Ogg, Opus, and WAV files with automatic metadata extraction
- **Streaming servers** - connect to Subsonic-compatible and Jellyfin servers alongside your local library
- **Playback** - gapless playback via mpv with queue management, shuffle, repeat, and ReplayGain support
- **Scrobbling** - Last.fm and ListenBrainz integration with offline queue for missed scrobbles
- **Search** - full-text search across titles, artists, albums, and genres
- **Playlists** - create, reorder, and manage playlists mixing local and streamed tracks
- **Stats** - listening history with top artists, albums, and tracks over configurable time periods
- **Artist info** - biographies and metadata from Last.fm and MusicBrainz, cached locally
- **Keyboard shortcuts** - full keyboard navigation (space to play/pause, arrows to seek, etc.)
- **Desktop notifications** - track change notifications via D-Bus
- **Dark/light/system themes** - follows your desktop preference or set manually

## Architecture

```
frontend/          Svelte 5 UI (TypeScript)
  src/             Components, stores, theme
  bindings/        Auto-generated Wails RPC bindings
internal/
  library/         SQLite database, queries, migrations
  player/          mpv wrapper, queue, MPRIS D-Bus
  metadata/        Audio file tag reading
  artistinfo/      Last.fm + MusicBrainz metadata fetching
  scrobbling/      Last.fm and ListenBrainz clients
  streaming/       Subsonic and Jellyfin API clients
  cue/             CUE sheet parser
  system/          Desktop notifications
libraryservice.go  Wails service: library operations
playerservice.go   Wails service: playback controls
main.go            Application entry point
```

The Go backend handles all I/O (database, network, playback). The Svelte frontend communicates via Wails RPC bindings. SQLite stores the library, playlists, play history, and configuration in a single `library.db` file.

## Building from source

### Prerequisites

- Go 1.25+
- Node.js 22+
- System libraries: GTK4, WebKitGTK 6.0, mpv, pkg-config
- [go-task](https://taskfile.dev) (or use `go install` directly)
- [Wails 3](https://wails.io): `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`

### Install with Nix

```sh
nix build github:willfish/forte
```

Or add to a NixOS/home-manager configuration:

```nix
environment.systemPackages = [
  (builtins.getFlake "github:willfish/forte").packages.${system}.default
];
```

### With Nix (development)

The repository includes a `flake.nix` that provides all dependencies:

```sh
# Enter the dev shell (or use direnv)
nix develop

# Build
task build

# Run in development mode (hot reload)
task dev
```

### Without Nix

Install the system dependencies for your distribution:

**Fedora/RHEL:**
```sh
sudo dnf install gtk4-devel webkitgtk6.0-devel mpv-devel pkg-config
```

**Debian/Ubuntu:**
```sh
sudo apt install libgtk-4-dev libwebkitgtk-6.0-dev libmpv-dev pkg-config
```

**Arch:**
```sh
sudo pacman -S gtk4 webkitgtk-6.0 mpv pkg-config
```

Then build:

```sh
cd frontend && npm ci && cd ..
task build
```

The binary is written to `bin/forte`.

## Demo mode

Seed the database with fixture data for screenshots and testing:

```sh
task demo
```

This creates 23 albums, 236 tracks, 3 playlists, and play history. Safe to run multiple times.

## Development

```sh
# Run with hot reload
task dev

# Run Go tests
go test -tags nocgo ./...

# Run frontend type checking
cd frontend && npm run check

# Run Playwright e2e tests (needs Chrome/Chromium)
cd frontend && CHROME_PATH=$(which google-chrome-stable) npm run test:e2e

# Lint
golangci-lint run --build-tags nocgo

# Security scan
govulncheck -tags nocgo ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch from `master`
3. Make your changes
4. Run tests and lint
5. Open a pull request

Issues and feature requests: [github.com/willfish/forte/issues](https://github.com/willfish/forte/issues)

## License

[GNU General Public License v3.0](LICENSE)
