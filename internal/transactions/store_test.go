package transactions

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
		CREATE TABLE transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			amount INTEGER NOT NULL,
			type TEXT NOT NULL DEFAULT 'expense',
			category TEXT NOT NULL DEFAULT 'other',
			description TEXT DEFAULT '',
			date TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
	`); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewStore(db), func() { db.Close() }
}

func TestAddList(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	id, err := s.Add(&core.Transaction{Amount: 1050, Type: "expense", Category: "food", Date: "2026-05-15"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero id")
	}

	txs, err := s.List(Filter{Year: 2026, Month: 5})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected 1 tx, got %d", len(txs))
	}
	if txs[0].Category != "food" {
		t.Errorf("expected food, got %s", txs[0].Category)
	}
	if txs[0].Amount != 1050 {
		t.Errorf("expected 1050, got %d", txs[0].Amount)
	}
}

func TestMonthSummary(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	s.Add(&core.Transaction{Amount: 50000, Type: "income", Category: "salary", Date: "2026-05-01"})
	s.Add(&core.Transaction{Amount: 1200, Type: "expense", Category: "food", Date: "2026-05-10"})
	s.Add(&core.Transaction{Amount: 3500, Type: "expense", Category: "food", Date: "2026-05-12"})

	summary, err := s.MonthSummary(2026, 5)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.Income != 50000 {
		t.Errorf("income: expected 50000, got %d", summary.Income)
	}
	if summary.Expense != 4700 {
		t.Errorf("expense: expected 4700, got %d", summary.Expense)
	}
	if summary.ByCategory["food"] != 4700 {
		t.Errorf("food category: expected 4700, got %d", summary.ByCategory["food"])
	}
}

func TestDelete(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	id, _ := s.Add(&core.Transaction{Amount: -500, Type: "expense", Category: "misc", Date: "2026-05-01"})
	if err := s.Delete(id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	txs, _ := s.List(Filter{Year: 2026, Month: 5})
	if len(txs) != 0 {
		t.Errorf("expected 0 txs after delete, got %d", len(txs))
	}
}

func TestDefaultDate(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	id, err := s.Add(&core.Transaction{Amount: -500, Type: "expense", Category: "misc"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	txs, _ := s.List(Filter{Year: time.Now().Year(), Month: int(time.Now().Month())})
	for _, tx := range txs {
		if tx.ID == id && tx.Date == "" {
			t.Error("expected default date to be set")
		}
	}
}
