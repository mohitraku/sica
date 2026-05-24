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
  models/         Domain types (Habit, HabitEntry). Zero dependencies.
  steward/        Business logic — completion/streak math, validation. No I/O.
  storage/        SQLite CRUD (modernc.org/sqlite, pure Go, no CGO).
                  HabitStore + EntryStore, concrete structs (no interfaces).
  tui/
    app.go        Bubble Tea Model — orchestrates storage ↔ views, handles
                  key/mouse dispatch with priority: confirm overlay →
                  form overlay → help → normal mode
    keys.go       Key bindings (keyMap with 12 bindings)
    views/        Stateless rendering components:
      habitlist.go  Scrollable list with click-to-toggle and streak display
      form.go       New/edit form (mode: FormNew/FormEdit)
      confirm.go    Yes/no delete confirmation overlay
      helpbar.go    Bottom bar with active key hints
      styles.go     Light/dark lipgloss styles, rebuilt on BackgroundColorMsg
```

**Data flow:** `cmd` creates stores → passes to `tui.Model` → model reads/writes via stores and passes data to `views` for rendering. Views never access storage directly. `steward` is imported by both `tui` and `views` for display logic (streak counts, completion status).

**SQLite:** WAL mode, FK enabled. Two tables: `habits` (PK id TEXT, name, frequency, target_value, quantity_type, timestamps) and `habit_entries` (PK id INTEGER, habit_id FK CASCADE, date, value, logged_at, UNIQUE(habit_id, date)). Schema is idempotent (`CREATE TABLE IF NOT EXISTS`), no versioned migrations.

**Data dir:** `SICA_DATA_DIR` env var, falls back to `~/.sica/`.
