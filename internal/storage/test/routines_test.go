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
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		routine_id  TEXT NOT NULL REFERENCES routines(id) ON DELETE CASCADE,
		date      TEXT NOT NULL,
		value     INTEGER NOT NULL DEFAULT 0,
		logged_at TEXT NOT NULL,
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

func sampleRoutine(id, name string) *models.Routine {
	now := models.NowUTC()
	return &models.Routine{
		ID:           id,
		Name:         name,
		Frequency:    "daily",
		TargetValue:  1,
		QuantityType: "binary",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestRoutineCreate(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewRoutineStore(db)

	r := sampleRoutine("a1b2c3d4e5f6a7b8", "Drink water")
	if err := store.Create(r); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.GetByID("a1b2c3d4e5f6a7b8")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil for existing routine")
	}
	if got.Name != "Drink water" {
		t.Errorf("expected name 'Drink water', got %q", got.Name)
	}
}

func TestRoutineGetByIDNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewRoutineStore(db)

	got, err := store.GetByID("nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent routine")
	}
}

func TestRoutineList(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewRoutineStore(db)

	if err := store.Create(sampleRoutine("id1", "First")); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.Create(sampleRoutine("id2", "Second")); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	routines, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(routines) != 2 {
		t.Errorf("expected 2 routines, got %d", len(routines))
	}
	// Most recent first
	if routines[0].Name != "Second" {
		t.Errorf("expected first routine to be 'Second', got %q", routines[0].Name)
	}
}

func TestRoutineListEmpty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewRoutineStore(db)

	routines, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(routines) != 0 {
		t.Errorf("expected 0 routines, got %d", len(routines))
	}
}

func TestRoutineUpdate(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewRoutineStore(db)

	r := sampleRoutine("id1", "Original")
	if err := store.Create(r); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	r.Name = "Updated"
	r.Frequency = "weekly"
	r.UpdatedAt = models.NowUTC()
	if err := store.Update(r); err != nil {
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

func TestRoutineUpdateNonexistent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewRoutineStore(db)

	r := sampleRoutine("nope", "Ghost")
	err := store.Update(r)
	if err == nil {
		t.Error("expected error updating nonexistent routine")
	}
}

func TestRoutineDelete(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewRoutineStore(db)

	if err := store.Create(sampleRoutine("id1", "ToDelete")); err != nil {
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

func TestRoutineDeleteNonexistent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewRoutineStore(db)

	err := store.Delete("nope")
	if err == nil {
		t.Error("expected error deleting nonexistent routine")
	}
}
