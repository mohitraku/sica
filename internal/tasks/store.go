package tasks

import (
	"database/sql"
	"time"

	"github.com/mojitrk/sica/internal/core"
	sqlstore "github.com/mojitrk/sica/internal/store/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(s *sqlstore.Store) *Store {
	return &Store{db: s.DB}
}

func (s *Store) CreateProject(p *core.Project) error {
	result, err := s.db.Exec(
		`INSERT INTO projects (name, description, color) VALUES (?, ?, ?)`,
		p.Name, p.Description, p.Color,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	p.ID = id
	return nil
}

func (s *Store) ListProjects() ([]core.Project, error) {
	rows, err := s.db.Query(
		`SELECT id, name, description, color FROM projects WHERE archived_at IS NULL ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []core.Project
	for rows.Next() {
		var p core.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Color); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

type Filter struct {
	Status    string
	Priority  string
	ProjectID *int64
}

func (s *Store) CreateTask(t *core.Task) error {
	t.CreatedAt = time.Now()
	result, err := s.db.Exec(
		`INSERT INTO tasks (title, description, status, priority, due_date, project_id)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		t.Title, t.Description, t.Status, t.Priority, t.DueDate, t.ProjectID,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	t.ID = id
	return nil
}

func scanTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", s)
}

func (s *Store) GetTask(id int64) (*core.Task, error) {
	t := &core.Task{}
	var dueDate, completedAt, createdAt sql.NullString
	err := s.db.QueryRow(
		`SELECT id, title, description, status, priority, due_date, project_id, created_at, completed_at
		 FROM tasks WHERE id=?`, id,
	).Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&dueDate, &t.ProjectID, &createdAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if createdAt.Valid {
		t.CreatedAt, _ = scanTime(createdAt.String)
	}
	if dueDate.Valid {
		d, _ := time.Parse("2006-01-02", dueDate.String)
		t.DueDate = &d
	}
	if completedAt.Valid {
		c, _ := scanTime(completedAt.String)
		t.CompletedAt = &c
	}
	return t, nil
}

func (s *Store) ListTasks(f Filter) ([]core.Task, error) {
	query := `SELECT id, title, description, status, priority, due_date, project_id, created_at, completed_at FROM tasks WHERE 1=1`
	var args []interface{}

	if f.Status != "" {
		query += ` AND status=?`
		args = append(args, f.Status)
	}
	if f.Priority != "" {
		query += ` AND priority=?`
		args = append(args, f.Priority)
	}
	if f.ProjectID != nil {
		query += ` AND project_id=?`
		args = append(args, *f.ProjectID)
	}

	query += ` ORDER BY
		CASE priority WHEN 'high' THEN 0 WHEN 'med' THEN 1 WHEN 'low' THEN 2 END,
		due_date ASC NULLS LAST,
		created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []core.Task
	for rows.Next() {
		var t core.Task
		var dueDate, completedAt, createdAt sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&dueDate, &t.ProjectID, &createdAt, &completedAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			t.CreatedAt, _ = scanTime(createdAt.String)
		}
		if dueDate.Valid {
			d, _ := time.Parse("2006-01-02", dueDate.String)
			t.DueDate = &d
		}
		if completedAt.Valid {
			c, _ := scanTime(completedAt.String)
			t.CompletedAt = &c
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *Store) UpdateTask(t *core.Task) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET title=?, description=?, status=?, priority=?, due_date=?, project_id=?
		 WHERE id=?`,
		t.Title, t.Description, t.Status, t.Priority, t.DueDate, t.ProjectID, t.ID,
	)
	return err
}

func (s *Store) CompleteTask(id int64) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET status='done', completed_at=datetime('now') WHERE id=?`, id,
	)
	return err
}

func (s *Store) DeleteTask(id int64) error {
	_, err := s.db.Exec(`DELETE FROM tasks WHERE id=?`, id)
	return err
}
