package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{DB: db}, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS habits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			frequency TEXT NOT NULL DEFAULT 'daily',
			target_value INTEGER NOT NULL DEFAULT 1,
			color TEXT DEFAULT '',
			icon TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			archived_at TEXT
		);

		CREATE TABLE IF NOT EXISTS habit_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			habit_id INTEGER NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
			date TEXT NOT NULL,
			value INTEGER NOT NULL DEFAULT 1,
			notes TEXT DEFAULT '',
			UNIQUE(habit_id, date)
		);

		CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			color TEXT DEFAULT '',
			archived_at TEXT
		);

		CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'todo',
			priority TEXT NOT NULL DEFAULT 'med',
			due_date TEXT,
			project_id INTEGER REFERENCES projects(id) ON DELETE SET NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			completed_at TEXT
		);

		CREATE TABLE IF NOT EXISTS ai_conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL DEFAULT 'New conversation',
			model TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS ai_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id INTEGER NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			tool_calls TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE INDEX IF NOT EXISTS idx_habit_entries_date ON habit_entries(date);
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	`)

	// Migration: add target_value for databases created before the column existed.
	// "duplicate column" error on fresh databases is expected and ignored.
	_, _ = db.Exec(`ALTER TABLE habits ADD COLUMN target_value INTEGER NOT NULL DEFAULT 1`)
	_, _ = db.Exec(`ALTER TABLE habits ADD COLUMN quantity_type TEXT NOT NULL DEFAULT 'count'`)

	return err
}
