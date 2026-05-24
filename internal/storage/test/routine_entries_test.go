package storage_test

import (
	"database/sql"
	"testing"

	"github.com/mohitraku/sica/internal/models"
	"github.com/mohitraku/sica/internal/storage"

	_ "modernc.org/sqlite"
)

func setupEntryTest(t *testing.T) (*storage.EntryStore, *storage.RoutineStore) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}
	if err := migrateTest(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	rStore := storage.NewRoutineStore(db)
	eStore := storage.NewEntryStore(db)

	now := models.NowUTC()
	r := &models.Routine{
		ID:           "routine1",
		Name:         "Test Routine",
		Frequency:    "daily",
		TargetValue:  1,
		QuantityType: "binary",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := rStore.Create(r); err != nil {
		t.Fatalf("Create routine failed: %v", err)
	}

	return eStore, rStore
}

func TestSetAndGetValue(t *testing.T) {
	eStore, _ := setupEntryTest(t)

	if err := eStore.SetValue("routine1", "2026-05-18", 1); err != nil {
		t.Fatalf("SetValue failed: %v", err)
	}

	val, err := eStore.GetValue("routine1", "2026-05-18")
	if err != nil {
		t.Fatalf("GetValue failed: %v", err)
	}
	if val != 1 {
		t.Errorf("expected value 1, got %d", val)
	}
}

func TestGetValueNoEntry(t *testing.T) {
	eStore, _ := setupEntryTest(t)

	val, err := eStore.GetValue("routine1", "2026-05-18")
	if err != nil {
		t.Fatalf("GetValue failed: %v", err)
	}
	if val != 0 {
		t.Errorf("expected value 0 for no entry, got %d", val)
	}
}

func TestSetValueUpsert(t *testing.T) {
	eStore, _ := setupEntryTest(t)

	if err := eStore.SetValue("routine1", "2026-05-18", 3); err != nil {
		t.Fatalf("first SetValue failed: %v", err)
	}
	if err := eStore.SetValue("routine1", "2026-05-18", 7); err != nil {
		t.Fatalf("second SetValue failed: %v", err)
	}

	val, _ := eStore.GetValue("routine1", "2026-05-18")
	if val != 7 {
		t.Errorf("expected updated value 7, got %d", val)
	}
}

func TestGetForRoutine(t *testing.T) {
	eStore, _ := setupEntryTest(t)

	eStore.SetValue("routine1", "2026-05-18", 1)
	eStore.SetValue("routine1", "2026-05-19", 0)

	entries, err := eStore.GetForRoutine("routine1")
	if err != nil {
		t.Fatalf("GetForRoutine failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Date != "2026-05-18" {
		t.Errorf("expected first date '2026-05-18', got %s", entries[0].Date)
	}
}

func TestGetForRoutineEmpty(t *testing.T) {
	eStore, _ := setupEntryTest(t)

	entries, err := eStore.GetForRoutine("routine1")
	if err != nil {
		t.Fatalf("GetForRoutine failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestEntryCascadeDelete(t *testing.T) {
	eStore, rStore := setupEntryTest(t)

	eStore.SetValue("routine1", "2026-05-18", 1)
	rStore.Delete("routine1")

	entries, _ := eStore.GetForRoutine("routine1")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after routine delete, got %d", len(entries))
	}
}
