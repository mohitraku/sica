package tasks

import (
	"testing"

	"github.com/mojitrk/sica/internal/core"
	"github.com/mojitrk/sica/internal/store/sqlite"
)

func setup(t *testing.T) *Store {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewStore(db)
}

func TestCreateAndListTasks(t *testing.T) {
	s := setup(t)

	err := s.CreateTask(&core.Task{Title: "Buy milk", Status: "todo", Priority: "med"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	tasks, err := s.ListTasks(Filter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "Buy milk" {
		t.Errorf("expected 'Buy milk', got '%s'", tasks[0].Title)
	}
}

func TestCompleteTask(t *testing.T) {
	s := setup(t)
	s.CreateTask(&core.Task{Title: "Write tests", Status: "todo", Priority: "high"})
	tasks, _ := s.ListTasks(Filter{})
	id := tasks[0].ID

	s.CompleteTask(id)
	task, _ := s.GetTask(id)
	if task.Status != "done" {
		t.Errorf("expected done, got %s", task.Status)
	}
	if task.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

func TestFilterByStatus(t *testing.T) {
	s := setup(t)
	s.CreateTask(&core.Task{Title: "Todo task", Status: "todo", Priority: "med"})
	s.CreateTask(&core.Task{Title: "Done task", Status: "done", Priority: "low"})

	tasks, _ := s.ListTasks(Filter{Status: "done"})
	if len(tasks) != 1 {
		t.Errorf("expected 1 done task, got %d", len(tasks))
	}
	if tasks[0].Title != "Done task" {
		t.Errorf("expected 'Done task', got '%s'", tasks[0].Title)
	}
}

func TestProjects(t *testing.T) {
	s := setup(t)
	s.CreateProject(&core.Project{Name: "Work"})
	projects, _ := s.ListProjects()
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].Name != "Work" {
		t.Errorf("expected 'Work', got '%s'", projects[0].Name)
	}
}

func TestDeleteTask(t *testing.T) {
	s := setup(t)
	s.CreateTask(&core.Task{Title: "Delete me", Status: "todo", Priority: "low"})
	tasks, _ := s.ListTasks(Filter{})
	s.DeleteTask(tasks[0].ID)
	tasks, _ = s.ListTasks(Filter{})
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks after delete, got %d", len(tasks))
	}
}
