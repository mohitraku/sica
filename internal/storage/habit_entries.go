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

func (s *EntryStore) SetValue(habitID, date string, value int) error {
	_, err := s.db.Exec(
		`INSERT INTO habit_entries (habit_id, date, value, logged_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(habit_id, date) DO UPDATE SET
		     value = excluded.value,
		     logged_at = excluded.logged_at`,
		habitID, date, value, models.NowUTC(),
	)
	return err
}

func (s *EntryStore) GetValue(habitID, date string) (int, error) {
	var value int
	err := s.db.QueryRow(
		`SELECT value FROM habit_entries WHERE habit_id = ? AND date = ?`,
		habitID, date,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return value, err
}

func (s *EntryStore) GetForHabit(habitID string) ([]models.HabitEntry, error) {
	rows, err := s.db.Query(
		`SELECT id, habit_id, date, value, logged_at
		 FROM habit_entries WHERE habit_id = ? ORDER BY date ASC`,
		habitID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.HabitEntry
	for rows.Next() {
		var e models.HabitEntry
		if err := rows.Scan(&e.ID, &e.HabitID, &e.Date, &e.Value, &e.LoggedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *EntryStore) GetForDateRange(habitID, from, to string) ([]models.HabitEntry, error) {
	rows, err := s.db.Query(
		`SELECT id, habit_id, date, value, logged_at
		 FROM habit_entries
		 WHERE habit_id = ? AND date >= ? AND date <= ?
		 ORDER BY date ASC`,
		habitID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.HabitEntry
	for rows.Next() {
		var e models.HabitEntry
		if err := rows.Scan(&e.ID, &e.HabitID, &e.Date, &e.Value, &e.LoggedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
