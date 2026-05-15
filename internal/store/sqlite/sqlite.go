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

		CREATE TABLE IF NOT EXISTS transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			amount INTEGER NOT NULL,
			type TEXT NOT NULL DEFAULT 'expense',
			category TEXT NOT NULL DEFAULT 'other',
			description TEXT DEFAULT '',
			date TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS budgets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category TEXT NOT NULL,
			amount_cents INTEGER NOT NULL,
			period TEXT NOT NULL DEFAULT 'monthly',
			start_date TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS calendar_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			location TEXT DEFAULT '',
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT 'local',
			outlook_id TEXT DEFAULT '',
			last_synced_at TEXT
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

		CREATE TABLE IF NOT EXISTS knowledge_docs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_path TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL DEFAULT '',
			source_url TEXT DEFAULT '',
			tags TEXT DEFAULT '[]',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS knowledge_chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			doc_id INTEGER NOT NULL REFERENCES knowledge_docs(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			embedding BLOB,
			chunk_index INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_habit_entries_date ON habit_entries(date);
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
		CREATE INDEX IF NOT EXISTS idx_calendar_events_start ON calendar_events(start_time);
		CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_doc ON knowledge_chunks(doc_id);
	`)

	// Migration: add target_value for existing databases.
	db.Exec(`ALTER TABLE habits ADD COLUMN target_value INTEGER NOT NULL DEFAULT 1`)

	return err
}
