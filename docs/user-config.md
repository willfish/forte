# User configuration file

Forte can export and import portable user settings as a sectioned TOML file.

## Location

- **Path:** `~/.config/forte/config.toml` (XDG config dir + `forte/`)
- **SQLite:** `~/.config/forte/library.db` (unchanged; still the runtime store)

## Format

- **Encoding:** TOML
- **`schemaVersion`:** top-level integer; current version is `1`
- **`exportedAt`:** RFC3339 UTC timestamp (set on export)

### Sections (schema v1)

| Section | TOML | Merge on import |
|---------|------|-----------------|
| App preferences | `[app]` | Overwrites `app_preferences` |
| Radio favourites | `[[radio.favourites]]` | Upsert by `stationUuid`; `pinned` from file wins |
| Custom stations | `[[radio.customStations]]` | Upsert by `stationUuid` |

Unknown top-level sections in a future file should be ignored (not implemented yet). Unsupported `schemaVersion` fails import with a clear error; startup logs the error and continues with existing DB state.

## Behaviour

- **Startup:** if `config.toml` exists, Forte imports it before loading preferences.
- **Settings:** Export / Import open a save/open dialog (TOML filter). Export writes the current DB state; Import applies the chosen file and refreshes app preferences in the UI.

## Adding a new section

1. Add structs in `internal/userconfig/config.go` and bump `SchemaVersion` when incompatible.
2. Export in `BuildFromDB` and import in `ApplyToDB` (or a dedicated `importX` helper).
3. Document merge rules in this file.
4. Add a round-trip test in `internal/userconfig/config_test.go`.
