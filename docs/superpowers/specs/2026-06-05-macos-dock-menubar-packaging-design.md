# macOS Dock Icon, Menubar Tray Parity, and Packaging Design

**Date**: 2026-06-05
**Status**: In design / ready for implementation with TDD
**Related**: Approach 1 selected (manual .app assembly in existing nix derivation)
**Skills applied**: brainstorming, test-driven-development, verification-before-completion, superpowers

## Problem Statement
- On macOS, installing via nix (e.g. `nix run github:willfish/forte`) results in the generic black "exec" default icon in the Dock instead of the Forte app icon.
- The menubar status item (equivalent to Linux tray) does not provide the same behaviour as Linux: primary click should show the control menu (not force window toggle), with "Show/Hide Window" in the menu performing the toggle.
- Users (including in ~/.dotfiles) are forced to wrap the package (symlinkJoin + DYLD_LIBRARY_PATH for libmpv, and vendorHash overrides) because the darwin package is a raw binary without bundle or runtime lib injection.
- Linux tray (colored theme icons, left-click toggle, playback state updates) must remain unchanged.

## Goals / Acceptance Criteria (AC)
All AC must be met and verified before claiming complete (per user instruction + verification-before-completion).

1. **Dock Icon**: When the darwin package is used on macOS, the app appears with the proper Forte icon in the Dock (from bundle .icns), not the default generic executable icon. The runtime `SetIcon` still works for updates if needed.
2. **Menubar Status Item Behaviour (B choice)**: On darwin:
   - Primary click on the menubar icon shows the control menu (Play/Pause, Stop, Next, Previous, Show/Hide Window, Quit).
   - "Show/Hide Window" menu item toggles the main window (using existing `AttachWindow`).
   - No custom `OnClick` that forces window toggle on click.
   - Menu contents identical to Linux.
3. **Menubar Icon Updates (parity)**: The menubar icon changes between "idle" and "playing" states when playback starts/stops (driven by `ps.onTrayUpdate`). Uses `SetTemplateIcon` for native macOS rendering (adapts to light/dark menu bar, system tint).
4. **Self-contained darwin Package (no wrapper needed)**: The flake's darwin package outputs a complete `Forte.app` bundle at `$out/Applications/Forte.app` such that:
   - `nix run` or profile install on mac gives a first-class app with icon.
   - The executable inside is wrapped to set `DYLD_LIBRARY_PATH` (or equivalent) including mpv's lib dir so go-mpv dlopen finds libmpv without user-side symlinkJoin or env.
   - `Info.plist` has correct `CFBundleIdentifier: io.github.willfish.forte`, `CFBundleIconFile`, executable name.
   - `.icns` generated from `build/appicon.png`.
5. **Linux Unchanged**: All existing Linux tray behaviour (colored 32px theme icons via `SetIcon`, left-click `OnClick` toggle, `onThemeChange`, full menu) continues to work exactly as before. No regressions in tests or packaging.
6. **TDD for Go Behaviour Changes**: All new/changed Go production code for macOS paths (tray setup, icon selection, MacOptions, crash log guard, darwin branches) is implemented strictly via TDD:
   - Write minimal failing test first.
   - Run test to verify correct failure (RED).
   - Write minimal code to pass (GREEN).
   - Verify all tests green + clean output.
   - Refactor only while green.
   - No production code written before its test.
7. **Crash Log Darwin Safe**: `setupCrashLog` no longer attempts Linux-only `/proc/self/maps` on darwin (graceful, no error).
8. **Wails Mac Config**: `application.New` sets `MacOptions` with `ActivationPolicyRegular` (app with window + menubar item stays running when window hidden).
9. **Build & Packaging Verification**:
   - `go test ./...` passes (including new tests).
   - `nix build .#packages.x86_64-darwin.default` (or aarch64) succeeds and produces valid bundle structure (verifiable with `ls`, `file`, `plutil` or strings on plist).
   - The inner binary in the bundle has the mpv wrapper.
   - No changes required to consumer configs like `~/.dotfiles/home/user/packages.nix` for basic mac use.
10. **No Regressions**: Existing Linux packaging (hicolor icons, desktop file, AppImage/deb/rpm), tests (tray_menu_test, etc.), and runtime behaviour unchanged.

## Scope
- In scope: Go runtime changes for darwin tray/Dock, new mac template icon assets (derived from existing SVGs), flake.nix darwin postInstall for .app assembly + icns + wrapper, minimal plist, TDD tests for Go behaviour, crash log fix, MacOptions.
- Out of scope: Full codesigning of the .app (nix limitation), new mac build tasks in Taskfile (unless trivial), changes to vendorHash (separate maintenance), UI/frontend changes, new tray themes for mac.
- Constraints: Keep Linux paths bit-for-bit identical. Use Approach 1 (manual bundle in existing go-build derivation, reuse wails3 for icns generation if possible). Prefer small PNG embeds for icons like existing tray assets. Use `makeWrapper` for the darwin launcher.

## Design Sections (Approved in Conversation)

### Section 1: macOS application and SystemTray initialisation (main.go)
(See previous design presentation for details: runtime.GOOS guards, MacOptions{ActivationPolicyRegular}, conditional SetTemplateIcon + no OnClick on darwin vs current Linux, update onThemeChange and onTrayUpdate, guard /proc in setupCrashLog.)

TDD note: Extract testable `chooseTrayIconForPlatform(...)` or extend `trayIconState` with platform-aware methods. Tests will cover darwin branches using build tags or helper funcs (linux host can test linux path; darwin build verifies other).

### Section 2: macOS menubar template icon assets
- Add `build/tray-macos-idle.png` and `build/tray-macos-playing.png` (32x32, monochrome black with alpha, derived from tray-*.svg via rsvg + magick for template compatibility).
- Embed in main.go like other tray icons.
- Selection in darwin path uses these based on playbackState only (theme ignored for menubar).
- Generated assets committed (like other tray-*-32.png).

### Section 3: darwin .app bundle and mpv wrapper in flake.nix (Approach 1)
- In the `forte` package derivation, add `postInstall` for `stdenv.isDarwin`:
  - Generate `icon.icns` from `build/appicon.png` (use wails3 icons command or magick + icns tooling available in darwin stdenv; fall back to script).
  - `mkdir -p $out/Applications/Forte.app/Contents/{MacOS,Resources}`
  - Write `Contents/Info.plist` (minimal XML with CFBundleIdentifier, CFBundleExecutable: "forte", CFBundleIconFile: "icon", NSPrincipalClass etc. as needed for Wails/Go app).
  - Copy (or link) the real binary.
  - Use `makeWrapper` (add to nativeBuildInputs) to create launcher at `Contents/MacOS/forte`:
    ```
    wrapProgram ... --prefix DYLD_LIBRARY_PATH : "${lib.makeLibraryPath [ mpv ]}"
    ```
    (Adjust for .app layout; the wrapper script execs the real binary from store or sibling.)
  - The top-level `bin/forte` can remain the real binary or point to launcher for convenience.
- Update `meta` if needed.
- The bundle makes `nix run` produce an app that macOS treats with proper icon and can be moved to /Applications.

### Section 4: Verification, tests, and rollout
- Go: Add unit tests for icon selection / platform branches (TDD driven). Run with `go test -tags=...` or build-tagged test files.
- Nix: After build for darwin target, verify:
  - `test -d result/Applications/Forte.app`
  - `test -f .../Contents/Resources/icon.icns`
  - `test -f .../Contents/Info.plist` (validate with plutil or grep for keys)
  - `file .../Contents/MacOS/forte` shows script or wrapper
  - Strings or otool show the DYLD prefix or wrapper content includes mpv path.
- Full `go test ./...` (linux paths).
- Update README.md with macOS install note (nix run works, icon and tray behave as expected).
- No user-facing config changes needed.

## Implementation Order (TDD + Red-Green-Refactor)
1. Add the two mac template PNG assets (generated).
2. TDD the icon selection helper / trayIconState extensions (test first for darwin vs linux bytes).
3. TDD the main.go changes: MacOptions, crash log guard, tray setup conditionals, listener updates (tests for branches where possible).
4. TDD any extracted testable units.
5. Update flake.nix for darwin postInstall / bundle (verify with nix build for darwin target).
6. Update docs/README if needed.
7. Run full verification (tests, nix darwin build + bundle inspection, simulate AC).
8. Refactor only while green.

## Risks / Open
- Generating .icns in pure darwin nix derivation: use wails3 (already in closure) `wails3 generate icons` or equivalent; test in build.
- Wrapper inside .app: make sure the launcher is the entry point in plist, and real binary is accessible (use absolute store path in wrapper).
- Cross-build from linux to darwin .app: possible for structure (icons, plist, binary cross-compiles fine); full macOS launch test requires darwin host or VM.
- Template icon quality: the generated 1-bit/gray may need visual tweak; committed and can be iterated.
- Existing tray_menu_test.go etc. must stay green.

## Verification Before Completion
Will use `verification-before-completion` skill + explicit commands:
- `go test ./... -count=1`
- Nix build for darwin target + `ls -R result/Applications/Forte.app` + plist validation + file checks.
- Confirm no diff in linux build outputs/behaviour.
- Run any manual simulation possible (e.g. strings on binary).
- All AC checked off with evidence.

This plan captures the agreed design from conversation. Implementation will strictly follow TDD for code and verification gate at end.

(End of plan)
