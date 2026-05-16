package budgets

import (
	"database/sql"

	"github.com/mojitrk/sica/internal/core"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(b *core.Budget) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO budgets (category, amount_cents, period, start_date) VALUES (?, ?, ?, ?)`,
		b.Category, b.AmountCents, b.Period, b.StartDate,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) List() ([]core.Budget, error) {
	rows, err := s.db.Query(
		`SELECT id, category, amount_cents, period, start_date FROM budgets ORDER BY category`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []core.Budget
	for rows.Next() {
		var b core.Budget
		if err := rows.Scan(&b.ID, &b.Category, &b.AmountCents, &b.Period, &b.StartDate); err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}
	return budgets, rows.Err()
}

func (s *Store) Update(b *core.Budget) error {
	_, err := s.db.Exec(
		`UPDATE budgets SET category = ?, amount_cents = ?, period = ?, start_date = ? WHERE id = ?`,
		b.Category, b.AmountCents, b.Period, b.StartDate, b.ID,
	)
	return err
}

func (s *Store) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM budgets WHERE id = ?`, id)
	return err
}
