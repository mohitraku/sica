package storage

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(dataDir string) (*sql.DB, error) {
	dir := dataDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dir, "sica.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS routines (
			id            TEXT NOT NULL PRIMARY KEY,
			name          TEXT NOT NULL,
			frequency     TEXT NOT NULL DEFAULT 'daily',
			target_value  INTEGER NOT NULL DEFAULT 1,
			quantity_type TEXT NOT NULL DEFAULT 'binary',
			created_at    TEXT NOT NULL,
			updated_at    TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS routine_entries (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			routine_id TEXT NOT NULL REFERENCES routines(id) ON DELETE CASCADE,
			date       TEXT NOT NULL,
			value      INTEGER NOT NULL DEFAULT 0,
			logged_at  TEXT NOT NULL,
			UNIQUE(routine_id, date)
		);

		CREATE INDEX IF NOT EXISTS idx_routine_entries_date ON routine_entries(date);

		CREATE TABLE IF NOT EXISTS tasks (
			id         TEXT NOT NULL PRIMARY KEY,
			title      TEXT NOT NULL,
			done       INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);

		`
	_, err := db.Exec(schema)
	return err
}
