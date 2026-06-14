# Plan 003: Reconcile deleted local tracks during library scans

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report. When done, update the status row for this plan in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat f32f673..HEAD -- internal/library/scanner.go internal/library/scanner_test.go internal/library/watcher.go internal/library/watcher_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: plans/001-align-local-verification-commands.md
- **Category**: bug
- **Planned at**: commit `f32f673`, 2026-06-14

## Why this matters

The scanner is responsible for the local music library database. It currently
adds or updates files found during a scan, but it does not remove local tracks
that no longer exist under the scanned directories. The watcher has removal
handling, but that only works while the app is running and receiving filesystem
events. A manual or startup scan after offline deletions can leave dead tracks
visible and playable.

## Current state

- `internal/library/scanner.go:43-69` collects all audio paths under the input
  directories.
- `internal/library/scanner.go:74-115` processes those paths in batches and
  returns without reconciling database rows not seen in the scan.
- `internal/library/scanner.go:251-275` upserts a changed file by deleting and
  reinserting the track at the same `file_path`.
- `internal/library/watcher.go:166-177` removes tracks only when it sees a
  remove/rename event for an audio path.
- `internal/library/scanner_test.go` covers empty dirs, real files, changed
  file updates, cancellation, progress, and nonexistent dirs, but no test covers
  deletion reconciliation after a later full scan.

Repo conventions: scanner tests use `openTestDB(t)`, `NewScanner(db)`, and
real generated audio fixtures where tag reading matters. For deletion
reconciliation, plain inserted DB rows are sufficient unless you need tag reads.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Scanner tests | `go test -tags nocgo ./internal/library -run 'TestScan|TestWatcher'` | all tests pass |
| Library package | `go test -tags nocgo ./internal/library` | all tests pass |
| Full Go tests | `go test -tags nocgo ./...` | all tests pass in full dev env |

## Scope

**In scope**:
- `internal/library/scanner.go`
- `internal/library/scanner_test.go`
- `internal/library/watcher.go` only if you extract a shared deletion helper
- `internal/library/watcher_test.go` only if the shared helper changes watcher
  behavior

**Out of scope**:
- Remote server sync reconciliation in `internal/library/sync.go`; it already
  has server-specific reconcile logic.
- Playlist/history policy decisions beyond local track deletion behavior.
- UI scan controls; those belong to plan 004.

## Git workflow

- Branch: `BAU-reconcile-local-scans`
- Do not push or open a PR unless instructed.

## Steps

### Step 1: Add a failing scanner deletion test

Add a test to `internal/library/scanner_test.go` that:

1. Creates a temporary scan directory.
2. Inserts a local track row whose `file_path` points inside that directory but
   does not exist on disk.
3. Inserts a second local track row outside the scanned directory.
4. Runs `scanner.Scan(context.Background(), []string{dir}, nil)`.
5. Asserts the missing in-scope track was removed and the out-of-scope track
   remains.

Also include a server track (`file_path` like `server://srv/track`) or
`server_id != ''` if the test setup is simple, and assert it is not removed by
local scan reconciliation.

**Verify**: `go test -tags nocgo ./internal/library -run TestScan` -> the new
test fails before implementation for the expected stale-track reason.

### Step 2: Implement local stale-track reconciliation

In `Scanner.Scan`, after successful batch processing, reconcile local DB tracks
whose `file_path` is under one of the scanned directories and not in the set of
paths collected at the start of the scan. Remove associated FTS entries before
deleting tracks, matching watcher behavior.

Use path-aware containment:

- Clean scanned dirs and track paths.
- Only reconcile local filesystem tracks (`server_id = ''` or `file_path NOT
  LIKE 'server://%'`).
- Do not delete rows outside the scanned roots.
- Prefer a transaction for the reconciliation step so FTS and track deletion do
  not drift.

If shared code helps, extract a small unexported helper for deleting a track by
ID or path and use it from both scanner and watcher.

**Verify**: `go test -tags nocgo ./internal/library -run TestScan` -> all scan
tests pass.

### Step 3: Preserve watcher behavior

If watcher code changed, run watcher tests and confirm direct remove handling
still works.

**Verify**: `go test -tags nocgo ./internal/library -run TestWatcher` -> all
watcher tests pass.

## Test plan

- New scanner deletion reconciliation test in `internal/library/scanner_test.go`.
- Existing tests to model:
  - `TestScanEmptyDir`
  - `TestScanChangedRealAudioFileUpdatesExistingTrack`
  - `TestWatcherDetectsRemove`
- Verification:
  - `go test -tags nocgo ./internal/library`
  - `go test -tags nocgo ./...` in a full dev environment

## Done criteria

- [ ] Full scans remove missing local tracks under scanned roots
- [ ] Full scans do not remove local tracks outside scanned roots
- [ ] Full scans do not remove server tracks
- [ ] FTS entries are removed with stale tracks
- [ ] `go test -tags nocgo ./internal/library` exits 0
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report if:

- There is a product requirement to keep deleted local files in the library as
  offline placeholders.
- Correct reconciliation requires changing playlist/history semantics beyond
  deleting local track rows and their FTS entries.
- The scanner API has changed and no longer receives the scan roots.

## Maintenance notes

If future code adds multiple local library roots or user-configured directory
state, keep stale reconciliation scoped to the roots the user explicitly scans.
The reviewer should check path containment carefully.
