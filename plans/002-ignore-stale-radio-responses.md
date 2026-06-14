# Plan 002: Ignore stale radio browse and search responses

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report. When done, update the status row for this plan in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat f32f673..HEAD -- frontend/src/RadioView.svelte frontend/tests/radio.spec.ts frontend/tests/mocks/wails-runtime.ts`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: MED
- **Depends on**: plans/001-align-local-verification-commands.md
- **Category**: bug
- **Planned at**: commit `f32f673`, 2026-06-14

## Why this matters

Radio browse/search is the app's default first screen. It starts multiple async
requests for featured stations, filtered stations, search results, favourites,
custom stations, history, and icon proxying. Older requests can still assign
`stations` and `loading` after the user has already changed filters or typed a
new search, producing stale results under the wrong search/filter state.

## Current state

- `frontend/src/RadioView.svelte:235-264` loads featured stations and assigns
  `stations`/`loading` when the request finishes.
- `frontend/src/RadioView.svelte:280-296` loads filtered stations and assigns
  `stations`/`loading` when the request finishes.
- `frontend/src/RadioView.svelte:433-459` debounces search, then assigns
  `stations`/`loading` without checking whether the query is still current.
- `frontend/src/RadioView.svelte:486-555` can rapidly trigger new loads through
  clear/remove/tag/source/country/codec filter actions.
- `frontend/src/RadioView.svelte:785-790` starts four data loads on mount.

The existing style is local Svelte state and direct async functions in the
component; match that style instead of introducing a global data library.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Frontend typecheck | `cd frontend && npm run check` | exit 0 |
| Radio e2e tests | `cd frontend && npx playwright test radio.spec.ts` | all tests pass |
| Full frontend e2e | `cd frontend && npm run test:e2e` | all tests pass |

## Scope

**In scope**:
- `frontend/src/RadioView.svelte`
- `frontend/tests/radio.spec.ts`
- `frontend/tests/mocks/wails-runtime.ts` only if the test needs controllable
  response delays

**Out of scope**:
- Backend radio clients
- Visual redesign of the radio page
- Replacing the local state model with a new store/data-fetching library

## Git workflow

- Branch: `BAU-stale-radio-responses`
- Do not push or open a PR unless instructed.

## Steps

### Step 1: Add a browse/search request generation guard

In `RadioView.svelte`, introduce a module-local or component-local monotonic
counter for the station list request, for example `let stationLoadSeq = 0`.
Before each station-list fetch path (`loadFeatured`, `loadSomaFMFiltered`,
`loadFiltered`, debounced search), increment the counter and capture the value.
Only assign `stations` and `loading = false` if the captured value is still the
latest.

Keep favourites/custom/history loading independent unless you find evidence of
the same stale assignment bug in those tabs.

**Verify**: `cd frontend && npm run check` -> exits 0.

### Step 2: Make clear-search and filter transitions invalidate pending work

Ensure `clearSearch`, `clearFilters`, `filterByTag`, `filterBySource`,
`filterByCountry`, and `filterByCodec` invalidate any pending debounced search
before starting their new load. The current code already clears
`debounceTimer`; keep that behavior and add the request guard so any already
started promise cannot overwrite newer state.

**Verify**: `cd frontend && npm run check` -> exits 0.

### Step 3: Add an e2e regression test

Add a Playwright test in `frontend/tests/radio.spec.ts` that forces an older
radio search/browse request to resolve after a newer one. Use
`frontend/tests/mocks/wails-runtime.ts` to add test-only delay controls if
needed, following the existing `localStorage` test hook style around
`forte.failPlayRadioStation`.

The test should assert that the visible station list matches the newest user
input/filter, not the late response.

**Verify**: `cd frontend && npx playwright test radio.spec.ts` -> all radio
tests pass, including the new stale-response regression.

## Test plan

- New e2e test in `frontend/tests/radio.spec.ts`:
  - Start one slow station-list request.
  - Trigger a newer search/filter request.
  - Let the slow response finish.
  - Assert the UI still shows the newer request's result and `Loading
    stations...` is not stuck.
- Existing tests to model: use the style in `frontend/tests/radio.spec.ts` and
  the mock patterns in `frontend/tests/mocks/wails-runtime.ts`.

## Done criteria

- [ ] Late station-list promises cannot overwrite newer `stations`
- [ ] Late station-list promises cannot incorrectly clear/set `loading`
- [ ] Regression test fails before the fix and passes after it
- [ ] `cd frontend && npm run check` exits 0
- [ ] `cd frontend && npx playwright test radio.spec.ts` exits 0
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report if:

- `RadioView.svelte` has been split or rewritten and the cited functions no
  longer exist.
- Solving the bug appears to require backend API changes.
- Playwright cannot be made deterministic without broad test harness changes
  outside the scoped files.

## Maintenance notes

Future radio data loads that write to `stations` should use the same guard. The
reviewer should specifically look for all `stations =` and `loading = false`
sites in `RadioView.svelte`.
