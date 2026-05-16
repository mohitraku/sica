package habits

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mojitrk/sica/internal/core"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(h *core.Habit) error {
	if h.TargetValue <= 0 {
		h.TargetValue = 1
	}
	if h.Frequency == "" {
		h.Frequency = "daily"
	}
	h.CreatedAt = time.Now()
	result, err := s.db.Exec(
		`INSERT INTO habits (name, description, frequency, target_value, color, icon) VALUES (?, ?, ?, ?, ?, ?)`,
		h.Name, h.Description, h.Frequency, h.TargetValue, h.Color, h.Icon,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	h.ID = id
	return nil
}

func (s *Store) Get(id int64) (*core.Habit, error) {
	h := &core.Habit{}
	var createdAt string
	err := s.db.QueryRow(
		`SELECT id, name, description, frequency, target_value, color, icon, created_at
		 FROM habits WHERE id=? AND archived_at IS NULL`, id,
	).Scan(&h.ID, &h.Name, &h.Description, &h.Frequency, &h.TargetValue, &h.Color, &h.Icon, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	h.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return h, err
}

func (s *Store) List(includeArchived bool) ([]core.Habit, error) {
	query := `SELECT id, name, description, frequency, target_value, color, icon, created_at FROM habits`
	if !includeArchived {
		query += ` WHERE archived_at IS NULL`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []core.Habit
	for rows.Next() {
		var h core.Habit
		var createdAt string
		if err := rows.Scan(&h.ID, &h.Name, &h.Description, &h.Frequency, &h.TargetValue, &h.Color, &h.Icon, &createdAt); err != nil {
			return nil, err
		}
		h.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		habits = append(habits, h)
	}
	return habits, rows.Err()
}

func (s *Store) Update(h *core.Habit) error {
	_, err := s.db.Exec(
		`UPDATE habits SET name=?, description=?, frequency=?, target_value=?, color=?, icon=? WHERE id=?`,
		h.Name, h.Description, h.Frequency, h.TargetValue, h.Color, h.Icon, h.ID,
	)
	return err
}

func (s *Store) Archive(id int64) error {
	_, err := s.db.Exec(`UPDATE habits SET archived_at=datetime('now') WHERE id=?`, id)
	return err
}

func (s *Store) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM habits WHERE id=?`, id)
	return err
}

func (s *Store) SetEntry(habitID int64, date string, value int, notes string) error {
	_, err := s.db.Exec(
		`INSERT INTO habit_entries (habit_id, date, value, notes) VALUES (?, ?, ?, ?)
		 ON CONFLICT(habit_id, date) DO UPDATE SET value=?, notes=?`,
		habitID, date, value, notes, value, notes,
	)
	return err
}

func (s *Store) GetEntry(habitID int64, date string) (int, error) {
	var value int
	err := s.db.QueryRow(
		`SELECT value FROM habit_entries WHERE habit_id=? AND date=?`,
		habitID, date,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return value, err
}

func (s *Store) IncrementEntry(habitID int64, date string, delta int) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("increment entry: %w", err)
	}
	defer tx.Rollback()

	var current int
	err = tx.QueryRow(
		`SELECT value FROM habit_entries WHERE habit_id=? AND date=?`,
		habitID, date,
	).Scan(&current)
	if err == sql.ErrNoRows {
		current = 0
	} else if err != nil {
		return 0, fmt.Errorf("increment entry: %w", err)
	}

	newValue := current + delta
	if newValue < 0 {
		newValue = 0
	}

	_, err = tx.Exec(
		`INSERT INTO habit_entries (habit_id, date, value, notes) VALUES (?, ?, ?, '')
		 ON CONFLICT(habit_id, date) DO UPDATE SET value=?, notes=''`,
		habitID, date, newValue, newValue,
	)
	if err != nil {
		return 0, fmt.Errorf("increment entry: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("increment entry commit: %w", err)
	}
	return newValue, nil
}

func (s *Store) RemoveEntry(habitID int64, date string) error {
	_, err := s.db.Exec(`DELETE FROM habit_entries WHERE habit_id=? AND date=?`, habitID, date)
	return err
}


type Stats struct {
	Habit         core.Habit
	CurrentStreak int
	LongestStreak int
	TotalEntries  int
	TodayValue    int
	TodayDone     bool
}

func (s *Store) Stats(habitID int64) (*Stats, error) {
	h, err := s.Get(habitID)
	if err != nil || h == nil {
		return nil, err
	}

	stats := &Stats{Habit: *h}

	rows, err := s.db.Query(
		`SELECT date, value FROM habit_entries WHERE habit_id=? ORDER BY date DESC`,
		habitID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []string
	dateValues := make(map[string]int)
	for rows.Next() {
		var d string
		var v int
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
		dates = append(dates, d)
		dateValues[d] = v
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	stats.TotalEntries = len(dates)
	if stats.TotalEntries == 0 {
		return stats, nil
	}

	today := time.Now().Format("2006-01-02")
	stats.TodayValue = dateValues[today]
	stats.TodayDone = stats.TodayValue >= h.TargetValue

	dateHit := make(map[string]bool, len(dates))
	for _, d := range dates {
		dateHit[d] = dateValues[d] >= h.TargetValue
	}

	current := 0
	cursor := today
	for {
		if done, ok := dateHit[cursor]; ok && done {
			current++
		} else {
			break
		}
		t, _ := time.Parse("2006-01-02", cursor)
		cursor = t.AddDate(0, 0, -1).Format("2006-01-02")
	}
	stats.CurrentStreak = current

	longest := 0
	run := 0
	sortDateAsc := make([]string, len(dates))
	for i, d := range dates {
		sortDateAsc[len(dates)-1-i] = d
	}
	// Walk day-by-day from the earliest entry to today to catch gaps.
	if len(sortDateAsc) > 0 {
		cursor, _ := time.Parse("2006-01-02", sortDateAsc[0])
		end, _ := time.Parse("2006-01-02", today)
		for !cursor.After(end) {
			ds := cursor.Format("2006-01-02")
			if dateHit[ds] {
				run++
				if run > longest {
					longest = run
				}
			} else {
				run = 0
			}
			cursor = cursor.AddDate(0, 0, 1)
		}
	}
	stats.LongestStreak = longest
	if current > longest {
		stats.LongestStreak = current
	}

	return stats, nil
}
