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
	return NewStore(db.DB)
}

func TestCreateAndListHabits(t *testing.T) {
	s := setup(t)

	err := s.Create(&core.Habit{Name: "Exercise", Frequency: "daily"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	habits, err := s.List()
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

func TestQuantityTypeDefaults(t *testing.T) {
	s := setup(t)

	s.Create(&core.Habit{Name: "Test", Frequency: "daily"})
	habits, _ := s.List()
	if habits[0].QuantityType != "count" {
		t.Errorf("expected default quantity_type 'count', got '%s'", habits[0].QuantityType)
	}

	s.Create(&core.Habit{Name: "BinHabit", Frequency: "daily", QuantityType: "binary", TargetValue: 5})
	habits, _ = s.List()
	for _, h := range habits {
		if h.Name == "BinHabit" {
			if h.TargetValue != 1 {
				t.Errorf("binary habit should force TargetValue=1, got %d", h.TargetValue)
			}
			if h.QuantityType != "binary" {
				t.Errorf("expected 'binary', got '%s'", h.QuantityType)
			}
		}
	}
}

func TestAddEntryAndStats(t *testing.T) {
	s := setup(t)
	s.Create(&core.Habit{Name: "Meditate", Frequency: "daily"})
	habits, _ := s.List()
	id := habits[0].ID

	today := time.Now().Format("2006-01-02")
	s.SetEntry(id, today, 1)

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

func TestTargetValueAndIncrement(t *testing.T) {
	s := setup(t)
	s.Create(&core.Habit{Name: "Water", Frequency: "daily", TargetValue: 8})
	habits, _ := s.List()
	id := habits[0].ID

	today := time.Now().Format("2006-01-02")

	newVal, err := s.IncrementEntry(id, today, 1)
	if err != nil {
		t.Fatalf("increment: %v", err)
	}
	if newVal != 1 {
		t.Errorf("expected 1, got %d", newVal)
	}

	newVal, err = s.IncrementEntry(id, today, 3)
	if err != nil {
		t.Fatalf("increment 3: %v", err)
	}
	if newVal != 4 {
		t.Errorf("expected 4, got %d", newVal)
	}

	newVal, err = s.IncrementEntry(id, today, -2)
	if err != nil {
		t.Fatalf("decrement 2: %v", err)
	}
	if newVal != 2 {
		t.Errorf("expected 2, got %d", newVal)
	}

	stats, _ := s.Stats(id)
	if stats.TodayValue != 2 {
		t.Errorf("TodayValue expected 2, got %d", stats.TodayValue)
	}
	if stats.TodayDone {
		t.Error("should not be done yet (2/8)")
	}

	s.SetEntry(id, today, 8)
	stats, _ = s.Stats(id)
	if !stats.TodayDone {
		t.Error("should be done now (8/8)")
	}
}

func TestBackfillEntry(t *testing.T) {
	s := setup(t)
	s.Create(&core.Habit{Name: "Read", Frequency: "daily", TargetValue: 1})
	habits, _ := s.List()
	id := habits[0].ID

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	s.SetEntry(id, yesterday, 1)

	val, _ := s.GetEntry(id, yesterday)
	if val != 1 {
		t.Errorf("expected backfill value 1, got %d", val)
	}
}

func TestDeleteHabit(t *testing.T) {
	s := setup(t)
	s.Create(&core.Habit{Name: "Delete me", Frequency: "daily"})
	habits, _ := s.List()
	s.Delete(habits[0].ID)
	habits, _ = s.List()
	if len(habits) != 0 {
		t.Errorf("expected 0 habits after delete, got %d", len(habits))
	}
}
