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
	CREATE TABLE IF NOT EXISTS habits (
		id            TEXT NOT NULL PRIMARY KEY,
		name          TEXT NOT NULL,
		frequency     TEXT NOT NULL DEFAULT 'daily',
		target_value  INTEGER NOT NULL DEFAULT 1,
		quantity_type TEXT NOT NULL DEFAULT 'binary',
		created_at    TEXT NOT NULL,
		updated_at    TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS habit_entries (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		habit_id  TEXT NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
		date      TEXT NOT NULL,
		value     INTEGER NOT NULL DEFAULT 0,
		logged_at TEXT NOT NULL,
		UNIQUE(habit_id, date)
	);

	CREATE INDEX IF NOT EXISTS idx_habit_entries_date ON habit_entries(date);
	`
	_, err := db.Exec(schema)
	return err
}
