# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & test

```bash
go build ./cmd/sica/          # build binary
go vet ./...                  # lint
go test ./...                 # all tests
go test -run TestX ./internal/steward/test/   # single test
```

Use `GOTOOLCHAIN=go1.25.0` if the system Go version is older.

## Release

Tag a `v*` tag and push — the GitHub Actions workflow in `.github/workflows/release.yml` runs GoReleaser. Platforms: linux/amd64, linux/arm64, windows/amd64. No macOS, no Homebrew.

```bash
git tag v0.2.0 && git push origin v0.2.0
```

Test the goreleaser config locally before tagging:
```bash
goreleaser release --clean --skip=publish --skip=validate
```

## Architecture

```
cmd/sica          Entry point — opens DB, creates stores, runs Bubble Tea loop
internal/
  models/         Domain types (Routine, RoutineEntry, Task, Person,
                  SignificantDate). Zero dependencies.
  steward/        Business logic — completion/streak math, validation. No I/O.
  storage/        SQLite CRUD (modernc.org/sqlite, pure Go, no CGO).
                  RoutineStore + EntryStore + TaskStore + PersonStore,
                  concrete structs.
  tui/
    app.go        Bubble Tea Model — orchestrates storage ↔ views, handles
                  key/mouse dispatch with priority: confirm overlay →
                  form overlay → help → settings → normal mode.
                  Three modes: ModeRoutines (default), ModeTasks, ModePeople
                  (Tab cycles through all three). Tab bar rendered above
                  title bar showing all modes.
    keys.go       Key bindings (keyMap with 17 bindings)
    views/        Stateless rendering components:
      routinelist.go  Scrollable list with click-to-toggle and streak display
      tasklist.go     Simple checklist with done/undone toggle
      peoplelist.go   People list with bump indicator and days-since-contact
      form.go         New/edit routine form (mode: FormNew/FormEdit)
      taskform.go     Single-field new/edit task form
      peopleform.go   Multi-field new/edit person form (name, email, phone)
      confirm.go      Yes/no delete confirmation overlay
      helpbar.go      Bottom bar with active key hints
      styles.go       Light/dark lipgloss styles, rebuilt on BackgroundColorMsg
      settings.go     Data directory configuration overlay
```

**Data flow:** `cmd` creates stores → passes to `tui.Model` → model reads/writes via stores and passes data to `views` for rendering. Views never access storage directly. `steward` is imported by both `tui` and `views` for display logic (streak counts, completion status, validation).

**SQLite:** WAL mode, FK enabled. Five tables: `routines` (PK id TEXT, name, frequency, target_value, quantity_type, timestamps), `routine_entries` (PK id INTEGER, routine_id FK CASCADE, date, value, logged_at, UNIQUE(routine_id, date)), `tasks` (PK id TEXT, title, done INTEGER, timestamps), `people` (PK id TEXT, name, email, phone, last_contacted, source, external_id, timestamps), and `significant_dates` (PK id INTEGER, person_id FK CASCADE, label, date, UNIQUE(person_id, label)). Schema is idempotent (`CREATE TABLE IF NOT EXISTS`), no versioned migrations.

**Data dir:** `SICA_DATA_DIR` env var, falls back to `~/.sica/`.
