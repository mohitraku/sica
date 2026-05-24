package storage

import (
	"database/sql"

	"github.com/mohitraku/sica/internal/models"
)

type EntryStore struct {
	db *sql.DB
}

func NewEntryStore(db *sql.DB) *EntryStore {
	return &EntryStore{db: db}
}

func (s *EntryStore) SetValue(routineID, date string, value int) error {
	_, err := s.db.Exec(
		`INSERT INTO routine_entries (routine_id, date, value, logged_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(routine_id, date) DO UPDATE SET
		     value = excluded.value,
		     logged_at = excluded.logged_at`,
		routineID, date, value, models.NowUTC(),
	)
	return err
}

func (s *EntryStore) GetValue(routineID, date string) (int, error) {
	var value int
	err := s.db.QueryRow(
		`SELECT value FROM routine_entries WHERE routine_id = ? AND date = ?`,
		routineID, date,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return value, err
}

func (s *EntryStore) GetForRoutine(routineID string) ([]models.RoutineEntry, error) {
	rows, err := s.db.Query(
		`SELECT id, routine_id, date, value, logged_at
		 FROM routine_entries WHERE routine_id = ? ORDER BY date ASC`,
		routineID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.RoutineEntry
	for rows.Next() {
		var e models.RoutineEntry
		if err := rows.Scan(&e.ID, &e.RoutineID, &e.Date, &e.Value, &e.LoggedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
