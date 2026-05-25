package storage

import (
	"database/sql"

	"github.com/mohitraku/sica/internal/models"
)

type PersonStore struct {
	db *sql.DB
}

func NewPersonStore(db *sql.DB) *PersonStore {
	return &PersonStore{db: db}
}

func (s *PersonStore) Create(p *models.Person) error {
	_, err := s.db.Exec(
		`INSERT INTO people (id, name, email, phone, last_contacted, source, external_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Email, p.Phone, p.LastContacted, p.Source, p.ExternalID, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (s *PersonStore) GetByID(id string) (*models.Person, error) {
	p := &models.Person{}
	err := s.db.QueryRow(
		`SELECT id, name, email, phone, last_contacted, source, external_id, created_at, updated_at
		 FROM people WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.Email, &p.Phone, &p.LastContacted, &p.Source, &p.ExternalID, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PersonStore) List() ([]models.Person, error) {
	rows, err := s.db.Query(
		`SELECT id, name, email, phone, last_contacted, source, external_id, created_at, updated_at
		 FROM people ORDER BY last_contacted = '' DESC, last_contacted ASC, name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var people []models.Person
	for rows.Next() {
		var p models.Person
		if err := rows.Scan(&p.ID, &p.Name, &p.Email, &p.Phone, &p.LastContacted, &p.Source, &p.ExternalID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		people = append(people, p)
	}
	return people, rows.Err()
}

func (s *PersonStore) Update(p *models.Person) error {
	result, err := s.db.Exec(
		`UPDATE people SET name = ?, email = ?, phone = ?, last_contacted = ?, source = ?, external_id = ?, updated_at = ?
		 WHERE id = ?`,
		p.Name, p.Email, p.Phone, p.LastContacted, p.Source, p.ExternalID, p.UpdatedAt, p.ID,
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

func (s *PersonStore) Delete(id string) error {
	result, err := s.db.Exec(`DELETE FROM people WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *PersonStore) GetSignificantDates(personID string) ([]models.SignificantDate, error) {
	rows, err := s.db.Query(
		`SELECT id, person_id, label, date FROM significant_dates
		 WHERE person_id = ? ORDER BY date ASC`,
		personID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []models.SignificantDate
	for rows.Next() {
		var d models.SignificantDate
		if err := rows.Scan(&d.ID, &d.PersonID, &d.Label, &d.Date); err != nil {
			return nil, err
		}
		dates = append(dates, d)
	}
	return dates, rows.Err()
}

func (s *PersonStore) SetSignificantDates(personID string, dates []models.SignificantDate) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM significant_dates WHERE person_id = ?`, personID); err != nil {
		return err
	}

	for _, d := range dates {
		if _, err := tx.Exec(
			`INSERT INTO significant_dates (person_id, label, date) VALUES (?, ?, ?)`,
			personID, d.Label, d.Date,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
