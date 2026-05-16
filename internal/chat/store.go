package chat

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

func (s *Store) CreateConversation(title, model string) (int64, error) {
	if title == "" {
		title = "New conversation"
	}
	result, err := s.db.Exec(
		`INSERT INTO ai_conversations (title, model) VALUES (?, ?)`,
		title, model,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) ListConversations() ([]core.Conversation, error) {
	rows, err := s.db.Query(
		`SELECT id, title, model, created_at FROM ai_conversations ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []core.Conversation
	for rows.Next() {
		var c core.Conversation
		var createdAt string
		if err := rows.Scan(&c.ID, &c.Title, &c.Model, &createdAt); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

func (s *Store) UpdateTitle(id int64, title string) error {
	_, err := s.db.Exec(`UPDATE ai_conversations SET title = ? WHERE id = ?`, title, id)
	return err
}

func (s *Store) DeleteConversation(id int64) error {
	_, err := s.db.Exec(`DELETE FROM ai_conversations WHERE id = ?`, id)
	return err
}

func (s *Store) AddMessage(convID int64, role, content string) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO ai_messages (conversation_id, role, content) VALUES (?, ?, ?)`,
		convID, role, content,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) Messages(convID int64) ([]core.Message, error) {
	rows, err := s.db.Query(
		`SELECT id, conversation_id, role, content, tool_calls, created_at
		 FROM ai_messages WHERE conversation_id = ? ORDER BY id ASC`,
		convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []core.Message
	for rows.Next() {
		var m core.Message
		var toolCalls sql.NullString
		var createdAt string
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &toolCalls, &createdAt); err != nil {
			return nil, err
		}
		if toolCalls.Valid {
			m.ToolCalls = toolCalls.String
		}
		m.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
