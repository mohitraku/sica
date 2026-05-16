package calendar

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/mojitrk/sica/internal/core"
)

func setup(t *testing.T) (*Store, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`
		CREATE TABLE calendar_events (
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
		CREATE INDEX IF NOT EXISTS idx_calendar_events_start ON calendar_events(start_time);
	`); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewStore(db), func() { db.Close() }
}

func TestCreateList(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	start := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)

	id, err := s.Create(&core.CalendarEvent{
		Title: "Team standup", Location: "Zoom",
		StartTime: start, EndTime: end,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero id")
	}

	events, err := s.List(
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "Team standup" {
		t.Errorf("expected Team standup, got %s", events[0].Title)
	}
	if events[0].Source != "local" {
		t.Errorf("expected local source, got %s", events[0].Source)
	}
}

func TestListDateRange(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	s.Create(&core.CalendarEvent{
		Title:     "May event",
		StartTime: time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC),
	})
	s.Create(&core.CalendarEvent{
		Title:     "June event",
		StartTime: time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC),
	})

	events, _ := s.List(
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	)
	if len(events) != 1 {
		t.Fatalf("expected 1 event in May, got %d", len(events))
	}
	if events[0].Title != "May event" {
		t.Errorf("expected May event, got %s", events[0].Title)
	}
}

func TestUpcoming(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future1 := now.Add(1 * time.Hour)
	future2 := now.Add(2 * time.Hour)

	s.Create(&core.CalendarEvent{
		Title: "Past event", StartTime: past, EndTime: past.Add(30*time.Minute),
	})
	s.Create(&core.CalendarEvent{
		Title: "Future event 1", StartTime: future1, EndTime: future1.Add(30*time.Minute),
	})
	s.Create(&core.CalendarEvent{
		Title: "Future event 2", StartTime: future2, EndTime: future2.Add(30*time.Minute),
	})

	events, err := s.Upcoming(10)
	if err != nil {
		t.Fatalf("upcoming: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 upcoming events, got %d", len(events))
	}
	if events[0].Title != "Future event 1" {
		t.Errorf("expected Future event 1, got %s", events[0].Title)
	}
}

func TestUpdate(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	start := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	id, _ := s.Create(&core.CalendarEvent{Title: "Old", StartTime: start, EndTime: end})

	err := s.Update(&core.CalendarEvent{
		ID: id, Title: "Updated", Description: "desc",
		StartTime: start, EndTime: end,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	events, _ := s.List(
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	)
	if events[0].Title != "Updated" {
		t.Errorf("expected Updated, got %s", events[0].Title)
	}
}

func TestDelete(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	start := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	id, _ := s.Create(&core.CalendarEvent{Title: "To delete", StartTime: start, EndTime: start.Add(time.Hour)})

	if err := s.Delete(id); err != nil {
		t.Fatalf("delete: %v", err)
	}

	events, _ := s.List(
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	)
	if len(events) != 0 {
		t.Errorf("expected 0 events after delete, got %d", len(events))
	}
}
