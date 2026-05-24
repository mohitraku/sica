package storage_test

import (
	"database/sql"
	"testing"

	"github.com/mohitraku/sica/internal/models"
	"github.com/mohitraku/sica/internal/storage"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}
	if err := migrateTest(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func migrateTest(db *sql.DB) error {
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

func sampleHabit(id, name string) *models.Habit {
	now := models.NowUTC()
	return &models.Habit{
		ID:           id,
		Name:         name,
		Frequency:    "daily",
		TargetValue:  1,
		QuantityType: "binary",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestHabitCreate(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewHabitStore(db)

	h := sampleHabit("a1b2c3d4e5f6a7b8", "Drink water")
	if err := store.Create(h); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.GetByID("a1b2c3d4e5f6a7b8")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil for existing habit")
	}
	if got.Name != "Drink water" {
		t.Errorf("expected name 'Drink water', got %q", got.Name)
	}
}

func TestHabitGetByIDNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewHabitStore(db)

	got, err := store.GetByID("nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent habit")
	}
}

func TestHabitList(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewHabitStore(db)

	if err := store.Create(sampleHabit("id1", "First")); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.Create(sampleHabit("id2", "Second")); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	habits, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(habits) != 2 {
		t.Errorf("expected 2 habits, got %d", len(habits))
	}
	// Most recent first
	if habits[0].Name != "Second" {
		t.Errorf("expected first habit to be 'Second', got %q", habits[0].Name)
	}
}

func TestHabitListEmpty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewHabitStore(db)

	habits, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(habits) != 0 {
		t.Errorf("expected 0 habits, got %d", len(habits))
	}
}

func TestHabitUpdate(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewHabitStore(db)

	h := sampleHabit("id1", "Original")
	if err := store.Create(h); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	h.Name = "Updated"
	h.Frequency = "weekly"
	h.UpdatedAt = models.NowUTC()
	if err := store.Update(h); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := store.GetByID("id1")
	if got.Name != "Updated" {
		t.Errorf("expected 'Updated', got %q", got.Name)
	}
	if got.Frequency != "weekly" {
		t.Errorf("expected 'weekly', got %q", got.Frequency)
	}
}

func TestHabitUpdateNonexistent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewHabitStore(db)

	h := sampleHabit("nope", "Ghost")
	err := store.Update(h)
	if err == nil {
		t.Error("expected error updating nonexistent habit")
	}
}

func TestHabitDelete(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewHabitStore(db)

	if err := store.Create(sampleHabit("id1", "ToDelete")); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.Delete("id1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := store.GetByID("id1")
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestHabitDeleteNonexistent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewHabitStore(db)

	err := store.Delete("nope")
	if err == nil {
		t.Error("expected error deleting nonexistent habit")
	}
}
