package storage_test

import (
	"testing"

	"github.com/mohitraku/sica/internal/models"
	"github.com/mohitraku/sica/internal/storage"
)

func sampleTask(id, title string, done bool) *models.Task {
	now := models.NowUTC()
	return &models.Task{
		ID:        id,
		Title:     title,
		Done:      done,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestTaskCreate(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewTaskStore(db)

	task := sampleTask("a1b2c3d4e5f6a7b8", "Fix the sink", false)
	if err := store.Create(task); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.GetByID("a1b2c3d4e5f6a7b8")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil for existing task")
	}
	if got.Title != "Fix the sink" {
		t.Errorf("expected title 'Fix the sink', got %q", got.Title)
	}
	if got.Done {
		t.Error("expected task to not be done")
	}
}

func TestTaskGetByIDNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewTaskStore(db)

	got, err := store.GetByID("nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent task")
	}
}

func TestTaskList(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewTaskStore(db)

	if err := store.Create(sampleTask("id1", "First", false)); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.Create(sampleTask("id2", "Second", true)); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	tasks, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
	// Undone tasks first
	if tasks[0].Title != "First" || tasks[0].Done {
		t.Error("expected first task to be undone 'First'")
	}
	if tasks[1].Title != "Second" || !tasks[1].Done {
		t.Error("expected second task to be done 'Second'")
	}
}

func TestTaskListEmpty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewTaskStore(db)

	tasks, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestTaskUpdate(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewTaskStore(db)

	task := sampleTask("id1", "Original", false)
	if err := store.Create(task); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	task.Title = "Updated"
	task.Done = true
	task.UpdatedAt = models.NowUTC()
	if err := store.Update(task); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := store.GetByID("id1")
	if got.Title != "Updated" {
		t.Errorf("expected 'Updated', got %q", got.Title)
	}
	if !got.Done {
		t.Error("expected task to be done")
	}
}

func TestTaskUpdateNonexistent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewTaskStore(db)

	task := sampleTask("nope", "Ghost", false)
	err := store.Update(task)
	if err == nil {
		t.Error("expected error updating nonexistent task")
	}
}

func TestTaskDelete(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewTaskStore(db)

	if err := store.Create(sampleTask("id1", "ToDelete", false)); err != nil {
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

func TestTaskDeleteNonexistent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewTaskStore(db)

	err := store.Delete("nope")
	if err == nil {
		t.Error("expected error deleting nonexistent task")
	}
}

