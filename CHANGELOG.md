# Changelog

All notable changes to Forte are generated from Conventional Commit history.

## [Unreleased]


### Build
- **nix:** Pin dev shell to nixos 25.11
- **go:** Relax toolchain minimum
- **nix:** Drop cachix workflow



### CI
- **deps:** Bump trufflehog to 3.95.5
- **deps:** Bump checkout to 6.0.2
- **changelog:** Generate release notes from conventional commits



### Features
- **crash:** Preserve per-run diagnostics



### Fixes
- **radio:** Reduce idle ipc polling
- **library:** Improve local scans and go toolchain

## [1.0.0] - 2026-06-04


### Build
- **wails:** Migrate bindings to typescript
- **nix:** Add package derivation
- **nix:** Fix hashes and add update workflow



### CI
- Add pipeline
- **release:** Add binary release pipeline
- **deps:** Bump setup-go to 6
- **deps:** Bump checkout to 6
- **deps:** Bump golangci action to 9.2.0
- **deps:** Bump setup-node to 6
- **vm:** Add desktop validation checks
- **package:** Prove distro installs
- **release:** Verify installer checksums
- **macos:** Add build validation path
- **deps:** Bump upload-artifact to 7
- **deps:** Bump nix installer action to 22
- Add git-hooks pre-commit integration



### Chores
- Initial commit



### Dependencies
- **deps:** Bump go-git to 5.16.5
- **deps:** Bump devalue to 5.6.4
- **deps:** Bump go-billy to 5.9.0
- **deps:** Fix security dependency warnings
- **deps:** Clear go-git alerts
- **deps:** Enable dependabot cooldown
- **deps:** Bump fsnotify to 1.10.1
- **deps:** Roll up safe dependabot updates
- **deps:** Bump x/net to 0.54.0
- **deps:** Bump vite to 8.0.14
- **deps:** Update project dependencies



### Documentation
- Add comprehensive readme
- Update readme screenshot
- Replace demo image url
- Refresh readme
- Update readme image and description
- Update readme image



### Features
- Scaffold wails svelte nix app
- **player:** Integrate mpv purego playback
- **player:** Add transport controls
- **player:** Enable gapless playlists
- **player:** Add replaygain modes
- **cue:** Add cue sheet parser
- **library:** Add sqlite fts schema
- **metadata:** Add audio tag reader
- **library:** Add music folder scanner
- **library:** Add filesystem watcher
- **library:** Add full-text search
- **ui:** Scaffold three-panel shell
- **ui:** Add now-playing bar
- **library:** Add album browser
- **library:** Add album track list
- **playback:** Add keyboard shortcuts
- **theme:** Add dark and light modes
- **queue:** Implement playback queue
- **queue:** Add shuffle and repeat modes
- **playlists:** Add persistent playlists
- **queue:** Add reorderable queue panel
- **playback:** Persist playback state
- **system:** Add mpris notifications and tray
- **streaming:** Add subsonic client
- **streaming:** Add jellyfin client
- **library:** Unify server and local tracks
- **search:** Add debounced library search
- **streaming:** Handle degraded servers
- **scrobbling:** Add lastfm support
- **scrobbling:** Add listenbrainz support
- **scrobbling:** Queue offline scrobbles
- **stats:** Add listening statistics
- **metadata:** Fetch artist metadata
- **demo:** Add screenshot fixture seeder
- **radio:** Add radiobrowser client and favourites
- **radio:** Add browse search and favourites view
- **radio:** Integrate stream playback
- **radio:** Parse icy stream metadata
- **ui:** Add album art-forward layout
- **ui:** Add fullscreen now-playing view
- **ui:** Add view enter animations
- **ui:** Collapse sidebar responsively
- **branding:** Add app icon and dynamic tray
- **theme:** Establish design tokens
- **radio:** Add browse tag and source filters
- **radio:** Show curated top-voted stations
- **crash:** Log process maps for address resolution
- **radio:** Make forte radio-first
- **radio:** Keep history as tab
- **branding:** Improve iconography and demo
- **theme:** Switch palette to green
- **radio:** Add station double-click toggle
- **settings:** Add themes and vim navigation
- **theme:** Add transparency settings
- **logging:** Add configurable diagnostics
- **radio:** Add station detail links
- **radio:** Complete station detail acceptance
- **config:** Add sectioned toml import export
- **macos:** Add dock icon and tray parity



### Fixes
- **ui:** Switch to gtk4 webkitgtk 6
- **radio:** Restore somafm station artwork
- **radio:** Render station artwork in webview
- **player:** Prevent mpv thread safety crashes
- **player:** Close remaining mpv thread gaps
- **radio:** Guard purego null pointer crash
- **playback:** Batch polling to reduce webkit crashes
- **radio:** Harden errors and remove debug artifacts
- **playback:** Batch remaining ipc polling
- **sync:** Harden sync and frontend runtime
- **nix:** Repair launcher icon
- **security:** Resolve code scanning alerts
- **ui:** Repair desktop and radio controls
- **tray:** Set cosmic tray identity
- **tray:** Make cosmic icon visible
- **tray:** Use app-style cosmic icon
- **radio:** Stop and active station state
- **playback:** Stabilize pause and stop state
- **playback:** Clear mpv state on stop
- **a11y:** Improve accessibility and tray menu tests
- **library:** Harden runtime and tray cleanup
- **radio:** Improve station artwork fallbacks
- **radio:** Narrow nullable station detail links
- **radio:** Render station detail artwork
- **tray:** Restore window toggle
- **branding:** Default launcher icon to blue
- **release:** Stage appimage icon
- **release:** Tidy frontend dependencies



### Releases
- **release:** Prepare 1.0.0



### Tests
- **system:** Add integration coverage
- Add library player artistinfo coverage
- **library:** Add service integration tests
- **frontend:** Add playwright e2e coverage
- **radio:** Add shared station fixtures
- **radio:** Extend fixture and detail coverage

<!-- generated by git-cliff -->
