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

| Section | TOML | Startup import (`config.toml`) | Settings → Import from file |
|---------|------|-------------------------------|----------------------------|
| App preferences | `[app]` | Overwrites `app_preferences` | Overwrites `app_preferences` |
| Radio favourites | `[[radio.favourites]]` | **Merge** — insert only new `stationUuid` | **Replace** — upsert; file wins |
| Custom stations | `[[radio.customStations]]` | **Merge** — insert only new UUID | **Replace** — upsert; file wins |

**Merge** means stations already in the database are left unchanged (pins, names, and metadata are not rolled back by an older export). **Replace** is for restoring from a backup you explicitly chose.

Local music directories are configured in Settings and persisted in SQLite.
They are not exported to `config.toml` in schema v1.

Unknown top-level sections in a future file should be ignored (not implemented yet). Unsupported `schemaVersion` fails import with a clear error; startup logs the error and continues with existing DB state.

## Behaviour

- **Startup:** if `config.toml` exists, Forte merges radio sections then loads preferences.
- **Settings → Save to config directory:** writes `~/.config/forte/config.toml` (primary dotfiles workflow).
- **Settings → Export copy:** save-as elsewhere (backups, sharing).
- **Settings → Import from file:** upsert from the chosen path (restore / migrate).

## Adding a new section

1. Add structs in `internal/userconfig/config.go` and bump `SchemaVersion` when incompatible.
2. Export in `BuildFromDB` and import in `ApplyToDB` (or a dedicated `importX` helper).
3. Document merge rules in this file.
4. Add a round-trip test in `internal/userconfig/config_test.go`.
