package knowledge

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/mojitrk/sica/internal/core"
)

type Store struct {
	db      *sql.DB
	baseDir string
}

type frontmatter struct {
	Title     string   `yaml:"title"`
	SourceURL string   `yaml:"source_url"`
	Tags      []string `yaml:"tags"`
	CreatedAt string   `yaml:"created_at"`
}

func NewStore(db *sql.DB, baseDir string) *Store {
	os.MkdirAll(baseDir, 0755)
	return &Store{db: db, baseDir: baseDir}
}

func slug(title string) string {
	s := strings.ToLower(title)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		if r == ' ' || r == '_' {
			return '-'
		}
		return -1
	}, s)
	if len(s) > 64 {
		s = s[:64]
	}
	return strings.Trim(s, "-")
}

func (s *Store) Save(title, body, sourceURL string, tags []string) (*core.KnowledgeDoc, error) {
	now := time.Now()
	slugTitle := slug(title)
	ts := now.Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.md", ts, slugTitle)
	relPath := filename
	fullPath := filepath.Join(s.baseDir, filename)

	fm := frontmatter{
		Title:     title,
		SourceURL: sourceURL,
		Tags:      tags,
		CreatedAt: now.Format(time.RFC3339),
	}

	fmBytes, _ := yaml.Marshal(fm)
	content := fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), body)

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	tagJSON, _ := json.Marshal(tags)

	result, err := s.db.Exec(
		`INSERT INTO knowledge_docs (file_path, title, source_url, tags) VALUES (?, ?, ?, ?)`,
		relPath, title, sourceURL, string(tagJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("insert doc: %w", err)
	}

	id, _ := result.LastInsertId()

	s.db.Exec(
		`INSERT INTO knowledge_fts (rowid, title, content) VALUES (?, ?, ?)`,
		id, title, body,
	)

	return &core.KnowledgeDoc{
		ID:        id,
		FilePath:  relPath,
		Title:     title,
		SourceURL: sourceURL,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *Store) Get(id int64) (*core.KnowledgeDoc, error) {
	doc := &core.KnowledgeDoc{}
	var tagsJSON string
	var createdAt, updatedAt string
	err := s.db.QueryRow(
		`SELECT id, file_path, title, source_url, tags, created_at, updated_at FROM knowledge_docs WHERE id=?`,
		id,
	).Scan(&doc.ID, &doc.FilePath, &doc.Title, &doc.SourceURL, &tagsJSON, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(tagsJSON), &doc.Tags)
	doc.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	doc.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return doc, nil
}

func (s *Store) ReadContent(id int64) (string, error) {
	doc, err := s.Get(id)
	if err != nil || doc == nil {
		return "", err
	}
	fullPath := filepath.Join(s.baseDir, doc.FilePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return string(data), nil
}

func (s *Store) List() ([]core.KnowledgeDoc, error) {
	rows, err := s.db.Query(
		`SELECT id, file_path, title, source_url, tags, created_at, updated_at
		 FROM knowledge_docs ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDocs(rows)
}

func (s *Store) Search(query string) ([]core.KnowledgeDoc, error) {
	rows, err := s.db.Query(
		`SELECT d.id, d.file_path, d.title, d.source_url, d.tags, d.created_at, d.updated_at
		 FROM knowledge_fts f
		 JOIN knowledge_docs d ON d.id = f.rowid
		 WHERE knowledge_fts MATCH ?
		 ORDER BY rank
		 LIMIT 50`, query,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDocs(rows)
}

func (s *Store) Delete(id int64) error {
	doc, err := s.Get(id)
	if err != nil {
		return err
	}
	if doc != nil {
		os.Remove(filepath.Join(s.baseDir, doc.FilePath))
	}
	s.db.Exec(`DELETE FROM knowledge_fts WHERE rowid=?`, id)
	s.db.Exec(`DELETE FROM knowledge_docs WHERE id=?`, id)
	return nil
}

func scanDocs(rows *sql.Rows) ([]core.KnowledgeDoc, error) {
	var docs []core.KnowledgeDoc
	for rows.Next() {
		var d core.KnowledgeDoc
		var tagsJSON, createdAt, updatedAt string
		if err := rows.Scan(&d.ID, &d.FilePath, &d.Title, &d.SourceURL, &tagsJSON, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &d.Tags)
		d.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		d.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
		if d.Tags == nil {
			d.Tags = []string{}
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}
