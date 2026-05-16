package transactions

import (
	"database/sql"
	"time"

	"github.com/mojitrk/sica/internal/core"
)

type Store struct {
	db *sql.DB
}

type Filter struct {
	Year  int
	Month int
}

type MonthSummary struct {
	Income     int64
	Expense    int64
	ByCategory map[string]int64
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Add(tx *core.Transaction) (int64, error) {
	if tx.Date == "" {
		tx.Date = time.Now().Format("2006-01-02")
	}
	result, err := s.db.Exec(
		`INSERT INTO transactions (amount, type, category, description, date) VALUES (?, ?, ?, ?, ?)`,
		tx.Amount, tx.Type, tx.Category, tx.Description, tx.Date,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) List(filter Filter) ([]core.Transaction, error) {
	var rows *sql.Rows
	var err error

	if filter.Year > 0 && filter.Month > 0 {
		start := time.Date(filter.Year, time.Month(filter.Month), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		end := time.Date(filter.Year, time.Month(filter.Month)+1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		rows, err = s.db.Query(
			`SELECT id, amount, type, category, description, date, created_at
			 FROM transactions WHERE date >= ? AND date < ? ORDER BY date DESC, id DESC`,
			start, end,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, amount, type, category, description, date, created_at
			 FROM transactions ORDER BY date DESC, id DESC LIMIT 100`,
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []core.Transaction
	for rows.Next() {
		var tx core.Transaction
		var createdAt string
		if err := rows.Scan(&tx.ID, &tx.Amount, &tx.Type, &tx.Category, &tx.Description, &tx.Date, &createdAt); err != nil {
			return nil, err
		}
		tx.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

func (s *Store) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM transactions WHERE id = ?`, id)
	return err
}

func (s *Store) MonthSummary(year, month int) (*MonthSummary, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	end := time.Date(year, time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")

	rows, err := s.db.Query(
		`SELECT type, category, SUM(amount) as total
		 FROM transactions WHERE date >= ? AND date < ?
		 GROUP BY type, category`, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ms := &MonthSummary{ByCategory: make(map[string]int64)}
	for rows.Next() {
		var typ, cat string
		var total int64
		if err := rows.Scan(&typ, &cat, &total); err != nil {
			return nil, err
		}
		if typ == "income" {
			ms.Income += total
		} else {
			ms.Expense += total
		}
		ms.ByCategory[cat] += total
	}
	return ms, rows.Err()
}
