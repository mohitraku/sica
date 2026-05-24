package storage

import (
	"database/sql"

	"github.com/mohitraku/sica/internal/models"
)

type TaskStore struct {
	db *sql.DB
}

func NewTaskStore(db *sql.DB) *TaskStore {
	return &TaskStore{db: db}
}

func (s *TaskStore) Create(t *models.Task) error {
	_, err := s.db.Exec(
		`INSERT INTO tasks (id, title, done, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		t.ID, t.Title, t.Done, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (s *TaskStore) GetByID(id string) (*models.Task, error) {
	t := &models.Task{}
	err := s.db.QueryRow(
		`SELECT id, title, done, created_at, updated_at
		 FROM tasks WHERE id = ?`, id,
	).Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TaskStore) List() ([]models.Task, error) {
	rows, err := s.db.Query(
		`SELECT id, title, done, created_at, updated_at
		 FROM tasks ORDER BY done ASC, created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *TaskStore) Update(t *models.Task) error {
	result, err := s.db.Exec(
		`UPDATE tasks SET title = ?, done = ?, updated_at = ?
		 WHERE id = ?`,
		t.Title, t.Done, t.UpdatedAt, t.ID,
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

func (s *TaskStore) Delete(id string) error {
	result, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

