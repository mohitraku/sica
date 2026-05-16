package budgets

import (
	"database/sql"
	"testing"

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
		CREATE TABLE budgets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category TEXT NOT NULL,
			amount_cents INTEGER NOT NULL,
			period TEXT NOT NULL DEFAULT 'monthly',
			start_date TEXT NOT NULL
		);
	`); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewStore(db), func() { db.Close() }
}

func TestCreateList(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	id, err := s.Create(&core.Budget{Category: "food", AmountCents: 50000, Period: "monthly", StartDate: "2026-05-01"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero id")
	}

	budgets, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(budgets) != 1 {
		t.Fatalf("expected 1 budget, got %d", len(budgets))
	}
	if budgets[0].Category != "food" {
		t.Errorf("expected food, got %s", budgets[0].Category)
	}
	if budgets[0].AmountCents != 50000 {
		t.Errorf("expected 50000, got %d", budgets[0].AmountCents)
	}
}

func TestUpdate(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	id, _ := s.Create(&core.Budget{Category: "food", AmountCents: 50000, Period: "monthly", StartDate: "2026-05-01"})

	err := s.Update(&core.Budget{ID: id, Category: "groceries", AmountCents: 60000, Period: "monthly", StartDate: "2026-05-01"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	budgets, _ := s.List()
	if budgets[0].Category != "groceries" {
		t.Errorf("expected groceries, got %s", budgets[0].Category)
	}
	if budgets[0].AmountCents != 60000 {
		t.Errorf("expected 60000, got %d", budgets[0].AmountCents)
	}
}

func TestDelete(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	id, _ := s.Create(&core.Budget{Category: "food", AmountCents: 50000, Period: "monthly", StartDate: "2026-05-01"})
	if err := s.Delete(id); err != nil {
		t.Fatalf("delete: %v", err)
	}

	budgets, _ := s.List()
	if len(budgets) != 0 {
		t.Errorf("expected 0 budgets after delete, got %d", len(budgets))
	}
}
