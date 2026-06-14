# Plan 001: Align local verification commands with CI build tags

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report. When done, update the status row for this plan in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat f32f673..HEAD -- Makefile README.md Taskfile.yml .github/workflows/ci.yml .golangci.yml`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: dx
- **Planned at**: commit `f32f673`, 2026-06-14

## Why this matters

The repo has solid CI, but the local shortcut commands do not match the
documented and CI build tags. A contributor running `make test` or `make lint`
gets failures or different coverage from CI on a normal shell without GTK/WebKit
pkg-config packages. This makes every later fix harder to verify locally and
encourages bypassing the real commands.

## Current state

- `README.md` documents `go test -tags nocgo ./...`, `golangci-lint run
  --build-tags nocgo`, and `govulncheck -tags nocgo ./...`.
- `.github/workflows/ci.yml` runs `go test -tags nocgo ./...` in the test job
  and `golangci-lint-action` with `args: --build-tags nocgo` in the lint job.
- `Makefile` currently diverges:

```makefile
test:
	go test ./...

lint:
	golangci-lint run
	cd frontend && npm run check
```

- Local audit evidence: `go test -list . -tags nocgo ./...` failed at the root
  package in this shell because GTK/WebKit pkg-config libraries are missing, but
  package tests under `internal/...` listed successfully. `govulncheck -tags
  nocgo ./...` failed for the same root Wails/GTK package-loading reason.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Go tests | `go test -tags nocgo ./...` | exit 0 |
| Frontend typecheck | `cd frontend && npm run check` | exit 0 |
| Frontend build | `cd frontend && npm run build` | exit 0 |
| Lint | `golangci-lint run --build-tags nocgo` | exit 0 |
| Security | `govulncheck -tags nocgo ./...` | exit 0 |

## Scope

**In scope**:
- `Makefile`
- `README.md` only if command names/descriptions need clarification
- `Taskfile.yml` only if you decide to add a `task test` or `task check`

**Out of scope**:
- Application source code
- CI workflow rewrites unrelated to command parity
- Installing system packages in the repo

## Git workflow

- Branch: `BAU-align-local-verification`
- Commit per logical unit.
- Do not push or open a PR unless instructed.

## Steps

### Step 1: Fix Makefile command parity

Update `Makefile` so `make test` runs `go test -tags nocgo ./...` and
`make lint` runs `golangci-lint run --build-tags nocgo` before the existing
frontend check.

**Verify**: `make test` -> exits 0 in a fully provisioned dev shell. If the
current shell lacks GTK/WebKit pkg-config dependencies, it may fail with the
same Wails package-loading error recorded above; in that case run inside
`nix develop .#full` and continue only if it passes there.

### Step 2: Add an explicit all-in-one check if useful

If there is no existing one-command local check, add a `check` target to
`Makefile` that depends on `test` and `lint`, and consider adding frontend
build only if it stays reasonably fast. Keep it aligned with README and CI.

**Verify**: `make check` -> exits 0 in the full dev environment.

### Step 3: Clarify the README shortcut

If you add `make check`, add a short README development line that names it as
the local pre-PR verification shortcut while preserving the existing explicit
commands.

**Verify**: `rg -n "make check|go test -tags nocgo|golangci-lint run --build-tags nocgo" README.md Makefile`
-> shows the same tags in both places.

## Test plan

- No new unit tests are needed for Makefile-only changes.
- Verification is command-based: `make test`, `make lint`, and, if added,
  `make check`.

## Done criteria

- [ ] `make test` invokes `go test -tags nocgo ./...`
- [ ] `make lint` invokes `golangci-lint run --build-tags nocgo`
- [ ] Any new README shortcut matches the actual Makefile target
- [ ] Full dev environment verification exits 0
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report if:

- The repo has added another canonical verification tool since `f32f673`.
- `make test` still fails inside `nix develop .#full` for reasons unrelated to
  missing local GTK/WebKit pkg-config packages.
- Fixing verification requires application source changes.

## Maintenance notes

When CI build tags change, update the Makefile and README in the same PR. The
reviewer should check command parity, not just whether the commands pass once.
