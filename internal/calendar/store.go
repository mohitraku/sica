package calendar

import (
	"database/sql"
	"time"

	"github.com/mojitrk/sica/internal/core"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(e *core.CalendarEvent) (int64, error) {
	if e.Source == "" {
		e.Source = "local"
	}
	result, err := s.db.Exec(
		`INSERT INTO calendar_events (title, description, location, start_time, end_time, source)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		e.Title, e.Description, e.Location,
		e.StartTime.Format("2006-01-02 15:04:05"),
		e.EndTime.Format("2006-01-02 15:04:05"),
		e.Source,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) List(start, end time.Time) ([]core.CalendarEvent, error) {
	rows, err := s.db.Query(
		`SELECT id, title, description, location, start_time, end_time, source, outlook_id, last_synced_at
		 FROM calendar_events
		 WHERE start_time >= ? AND start_time < ?
		 ORDER BY start_time ASC, id ASC`,
		start.Format("2006-01-02 15:04:05"),
		end.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (s *Store) Upcoming(limit int) ([]core.CalendarEvent, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	rows, err := s.db.Query(
		`SELECT id, title, description, location, start_time, end_time, source, outlook_id, last_synced_at
		 FROM calendar_events
		 WHERE end_time >= ?
		 ORDER BY start_time ASC, id ASC
		 LIMIT ?`,
		now, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (s *Store) Update(e *core.CalendarEvent) error {
	_, err := s.db.Exec(
		`UPDATE calendar_events
		 SET title = ?, description = ?, location = ?, start_time = ?, end_time = ?
		 WHERE id = ?`,
		e.Title, e.Description, e.Location,
		e.StartTime.Format("2006-01-02 15:04:05"),
		e.EndTime.Format("2006-01-02 15:04:05"),
		e.ID,
	)
	return err
}

func (s *Store) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM calendar_events WHERE id = ?`, id)
	return err
}

func scanEvents(rows *sql.Rows) ([]core.CalendarEvent, error) {
	var events []core.CalendarEvent
	for rows.Next() {
		var e core.CalendarEvent
		var startStr, endStr string
		var outlookID sql.NullString
		var lastSyncedAt sql.NullString
		if err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.Location,
			&startStr, &endStr, &e.Source, &outlookID, &lastSyncedAt); err != nil {
			return nil, err
		}
		e.StartTime, _ = time.Parse("2006-01-02 15:04:05", startStr)
		e.EndTime, _ = time.Parse("2006-01-02 15:04:05", endStr)
		if outlookID.Valid {
			e.OutlookID = outlookID.String
		}
		if lastSyncedAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", lastSyncedAt.String)
			e.LastSyncedAt = &t
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
