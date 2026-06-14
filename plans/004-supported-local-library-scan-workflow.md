# Plan 004: Add a supported local library scan workflow

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report. When done, update the status row for this plan in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat f32f673..HEAD -- README.md docs/user-config.md libraryservice.go frontend/src/Settings.svelte frontend/src/Sidebar.svelte frontend/src/Content.svelte internal/library/scanner.go internal/library/watcher.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: L
- **Risk**: HIGH
- **Depends on**: plans/001-align-local-verification-commands.md, plans/003-reconcile-deleted-local-tracks.md
- **Category**: direction
- **Planned at**: commit `f32f673`, 2026-06-14

## Why this matters

The README says Forte can manage a local music library, but the shipped Wails
service surface and settings UI do not expose a supported way to choose local
music directories, start a scan, or monitor scan progress. The internal scanner
and watcher are present and tested, so this is a grounded product gap rather
than a speculative feature. Closing it makes the README promise true and turns
existing backend infrastructure into a usable workflow.

## Current state

- `README.md` says Forte can "scan local FLAC, MP3, Ogg, Opus, and WAV files,
  then browse albums, artists, tracks, playlists, and listening stats."
- `libraryservice.go` exposes library browsing methods such as `GetAlbums`,
  `GetAlbumTracks`, `Search`, playlist methods, and server sync, but no
  `ScanLibrary`, `AddMusicDirectory`, or equivalent Wails method exists.
- `frontend/src/Settings.svelte:614` renders "Library mode" preference, and
  `frontend/src/Settings.svelte:704` starts streaming-server settings; there is
  no local directory picker or scan action between those concerns.
- `internal/library/scanner.go:43-115` implements cancellable directory scans
  with optional progress.
- `internal/library/watcher.go:35-44` can watch configured directories, but no
  production service wires user-selected roots into it.
- `docs/user-config.md` documents app preferences, radio favourites, and custom
  stations only; no local music directory section exists.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Generate bindings | `task common:generate:bindings` | exit 0; frontend bindings update |
| Go library tests | `go test -tags nocgo ./internal/library` | all pass |
| Go root/service tests | `go test -tags nocgo ./...` | all pass in full dev env |
| Frontend typecheck | `cd frontend && npm run check` | exit 0 |
| Settings e2e | `cd frontend && npx playwright test settings.spec.ts` | all tests pass |

## Scope

**In scope**:
- `internal/library/*` files needed to persist local library roots
- `libraryservice.go`
- `frontend/src/Settings.svelte`
- Generated files under `frontend/bindings/`
- `frontend/tests/settings.spec.ts`
- `frontend/tests/mocks/wails-runtime.ts`
- `docs/user-config.md`
- `README.md` only if wording needs to match the final shipped scope

**Out of scope**:
- Replacing SQLite schema architecture
- Streaming server sync redesign
- Full library UI redesign outside the minimal controls needed to scan local
  directories
- Background scan scheduling beyond an explicit user-triggered scan, unless it
  is already straightforward from existing watcher code

## Git workflow

- Branch: `BAU-local-library-scan-workflow`
- Do not push or open a PR unless instructed.

## Steps

### Step 1: Decide and persist the local-directory model

Add a small database table or user-config section for local music directories.
Prefer SQLite if the app needs runtime mutation from Settings; optionally export
the same roots through `docs/user-config.md` later.

Add tests for add/list/remove behavior. Follow the style of
`internal/library/servers_test.go` for database-backed settings.

**Verify**: `go test -tags nocgo ./internal/library -run 'Test.*Directory|Test.*Music'`
-> new tests pass.

### Step 2: Expose Wails service methods

Add explicit methods to `LibraryService`, for example:

- `GetMusicDirectories() ([]string, error)`
- `AddMusicDirectory(path string) error`
- `RemoveMusicDirectory(path string) error`
- `ScanMusicLibrary() error` or `ScanMusicDirectory(path string) error`

Use `Dialogs.OpenFile`/directory selection from the frontend rather than
accepting arbitrary typed paths when possible. Keep backend validation anyway:
path exists, path is a directory, and duplicate roots are normalised.

**Verify**: `task common:generate:bindings` -> exits 0 and updates
`frontend/bindings/`.

### Step 3: Add Settings UI controls

In `frontend/src/Settings.svelte`, add a compact local-library section near the
existing "Library mode" preference:

- list configured directories
- add directory via native dialog
- remove directory
- scan now
- show busy/success/error state

Match existing Settings patterns for `userConfigBusy`, `syncing`, and result
messages. Do not add a landing-page-style explanation.

**Verify**: `cd frontend && npm run check` -> exits 0.

### Step 4: Add frontend mocks and e2e coverage

Update `frontend/tests/mocks/wails-runtime.ts` with the new LibraryService
methods. Add `settings.spec.ts` coverage for adding a directory, listing it,
triggering scan, and removing it. Use deterministic mock paths; do not require
real filesystem access in Playwright.

**Verify**: `cd frontend && npx playwright test settings.spec.ts` -> all
settings tests pass.

### Step 5: Update docs to match the shipped workflow

Update `docs/user-config.md` if local directories are exportable/importable.
Update README only if the implementation scope changes the current local
library claim.

**Verify**: `rg -n "local|music director|scan|Library mode" README.md docs/user-config.md frontend/src/Settings.svelte`
-> wording and UI terms are consistent.

## Test plan

- Backend tests for directory persistence and validation.
- Backend scan service test if feasible without Wails runtime.
- Frontend settings e2e test using Wails runtime mocks.
- Existing tests:
  - `go test -tags nocgo ./internal/library`
  - `go test -tags nocgo ./...`
  - `cd frontend && npm run check`
  - `cd frontend && npx playwright test settings.spec.ts`

## Done criteria

- [ ] A user can enable library mode, add a local directory, scan it, and remove
  it from Settings
- [ ] Local directory state persists across app restarts
- [ ] Wails bindings are regenerated and committed with the service changes
- [ ] Tests cover directory add/list/remove and scan trigger behavior
- [ ] Docs no longer overpromise or omit the supported local-library workflow
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report if:

- Plan 003 has not landed or scan reconciliation remains unresolved.
- Wails directory picker support is unavailable and implementing a safe fallback
  would require broad UI changes.
- The desired product direction is to remove local library support instead of
  shipping scan controls. In that case, replace this with a docs/cleanup plan.

## Maintenance notes

If the app later starts watching directories automatically, reuse the persisted
roots from this plan and make watcher lifecycle part of `LibraryService`
startup/shutdown. Reviewers should scrutinize path normalisation, deletion
behavior, and whether scan progress leaves the UI in a stuck busy state.
