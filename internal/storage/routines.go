package storage

import (
	"database/sql"

	"github.com/mohitraku/sica/internal/models"
)

type RoutineStore struct {
	db *sql.DB
}

func NewRoutineStore(db *sql.DB) *RoutineStore {
	return &RoutineStore{db: db}
}

func (s *RoutineStore) Create(r *models.Routine) error {
	_, err := s.db.Exec(
		`INSERT INTO routines (id, name, frequency, target_value, quantity_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.Frequency, r.TargetValue, r.QuantityType, r.CreatedAt, r.UpdatedAt,
	)
	return err
}

func (s *RoutineStore) GetByID(id string) (*models.Routine, error) {
	r := &models.Routine{}
	err := s.db.QueryRow(
		`SELECT id, name, frequency, target_value, quantity_type, created_at, updated_at
		 FROM routines WHERE id = ?`, id,
	).Scan(&r.ID, &r.Name, &r.Frequency, &r.TargetValue, &r.QuantityType, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *RoutineStore) List() ([]models.Routine, error) {
	rows, err := s.db.Query(
		`SELECT id, name, frequency, target_value, quantity_type, created_at, updated_at
		 FROM routines ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routines []models.Routine
	for rows.Next() {
		var r models.Routine
		if err := rows.Scan(&r.ID, &r.Name, &r.Frequency, &r.TargetValue, &r.QuantityType, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		routines = append(routines, r)
	}
	return routines, rows.Err()
}

func (s *RoutineStore) Update(r *models.Routine) error {
	result, err := s.db.Exec(
		`UPDATE routines SET name = ?, frequency = ?, target_value = ?, quantity_type = ?, updated_at = ?
		 WHERE id = ?`,
		r.Name, r.Frequency, r.TargetValue, r.QuantityType, r.UpdatedAt, r.ID,
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

func (s *RoutineStore) Delete(id string) error {
	result, err := s.db.Exec(`DELETE FROM routines WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
