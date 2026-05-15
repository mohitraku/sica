package habits

import (
	"testing"
	"time"

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

func TestCreateAndListHabits(t *testing.T) {
	s := setup(t)

	err := s.Create(&core.Habit{Name: "Exercise", Frequency: "daily"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	habits, err := s.List(false)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(habits) != 1 {
		t.Fatalf("expected 1 habit, got %d", len(habits))
	}
	if habits[0].Name != "Exercise" {
		t.Errorf("expected 'Exercise', got '%s'", habits[0].Name)
	}
}

func TestArchiveHidesHabit(t *testing.T) {
	s := setup(t)
	s.Create(&core.Habit{Name: "Read", Frequency: "daily"})
	habits, _ := s.List(false)
	id := habits[0].ID

	s.Archive(id)
	habits, _ = s.List(false)
	if len(habits) != 0 {
		t.Errorf("expected 0 active habits, got %d", len(habits))
	}
}

func TestAddEntryAndStats(t *testing.T) {
	s := setup(t)
	s.Create(&core.Habit{Name: "Meditate", Frequency: "daily"})
	habits, _ := s.List(false)
	id := habits[0].ID

	today := time.Now().Format("2006-01-02")
	s.AddEntry(id, today, 1, "")

	stats, err := s.Stats(id)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalEntries != 1 {
		t.Errorf("expected 1 entry, got %d", stats.TotalEntries)
	}
	if !stats.TodayDone {
		t.Error("expected today to be done")
	}
	if stats.CurrentStreak != 1 {
		t.Errorf("expected streak 1, got %d", stats.CurrentStreak)
	}
}

func TestDeleteHabit(t *testing.T) {
	s := setup(t)
	s.Create(&core.Habit{Name: "Delete me", Frequency: "daily"})
	habits, _ := s.List(false)
	s.Delete(habits[0].ID)
	habits, _ = s.List(false)
	if len(habits) != 0 {
		t.Errorf("expected 0 habits after delete, got %d", len(habits))
	}
}
