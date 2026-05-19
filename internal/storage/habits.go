package storage

import (
	"database/sql"

	"github.com/mohitraku/sica/internal/models"
)

type HabitStore struct {
	db *sql.DB
}

func NewHabitStore(db *sql.DB) *HabitStore {
	return &HabitStore{db: db}
}

func (s *HabitStore) Create(h *models.Habit) error {
	_, err := s.db.Exec(
		`INSERT INTO habits (id, name, frequency, target_value, quantity_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		h.ID, h.Name, h.Frequency, h.TargetValue, h.QuantityType, h.CreatedAt, h.UpdatedAt,
	)
	return err
}

func (s *HabitStore) GetByID(id string) (*models.Habit, error) {
	h := &models.Habit{}
	err := s.db.QueryRow(
		`SELECT id, name, frequency, target_value, quantity_type, created_at, updated_at
		 FROM habits WHERE id = ?`, id,
	).Scan(&h.ID, &h.Name, &h.Frequency, &h.TargetValue, &h.QuantityType, &h.CreatedAt, &h.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (s *HabitStore) List() ([]models.Habit, error) {
	rows, err := s.db.Query(
		`SELECT id, name, frequency, target_value, quantity_type, created_at, updated_at
		 FROM habits ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []models.Habit
	for rows.Next() {
		var h models.Habit
		if err := rows.Scan(&h.ID, &h.Name, &h.Frequency, &h.TargetValue, &h.QuantityType, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		habits = append(habits, h)
	}
	return habits, rows.Err()
}

func (s *HabitStore) Update(h *models.Habit) error {
	result, err := s.db.Exec(
		`UPDATE habits SET name = ?, frequency = ?, target_value = ?, quantity_type = ?, updated_at = ?
		 WHERE id = ?`,
		h.Name, h.Frequency, h.TargetValue, h.QuantityType, h.UpdatedAt, h.ID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *HabitStore) Delete(id string) error {
	result, err := s.db.Exec(`DELETE FROM habits WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
