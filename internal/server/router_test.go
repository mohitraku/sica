package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mojitrk/sica/internal/habits"
	"github.com/mojitrk/sica/internal/store/sqlite"
	"github.com/mojitrk/sica/internal/tasks"
)

func testDeps(t *testing.T) Deps {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return Deps{
		Store:       db,
		HabitsStore: habits.NewStore(db.DB),
		TasksStore:  tasks.NewStore(db.DB),
	}
}

func TestHealthEndpoint(t *testing.T) {
	handler := New(testDeps(t))
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAPIPlaceholders(t *testing.T) {
	handler := New(testDeps(t))
	endpoints := []string{
		"/api/habits/",
		"/api/tasks/",
		"/api/projects/",
		"/api/transactions/",
		"/api/budgets",
		"/api/summary",
		"/api/calendar/events",
		"/api/knowledge/docs",
		"/api/ai/conversations",
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest("GET", ep, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", ep, rec.Code)
		}
	}
}

func TestWebUI(t *testing.T) {
	handler := New(testDeps(t))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("web UI: expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("web UI: expected text/html, got %s", ct)
	}
}
