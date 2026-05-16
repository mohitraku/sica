package chat

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func setup(t *testing.T) (*Store, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`
		CREATE TABLE ai_conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL DEFAULT 'New conversation',
			model TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE ai_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id INTEGER NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			tool_calls TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewStore(db), func() { db.Close() }
}

func TestConversationCRUD(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	id, err := s.CreateConversation("Test chat", "qwen2.5:14b")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero id")
	}

	convs, err := s.ListConversations()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(convs) != 1 {
		t.Fatalf("expected 1 conv, got %d", len(convs))
	}
	if convs[0].Title != "Test chat" {
		t.Errorf("expected Test chat, got %s", convs[0].Title)
	}

	if err := s.UpdateTitle(id, "Updated title"); err != nil {
		t.Fatalf("update title: %v", err)
	}
	convs, _ = s.ListConversations()
	if convs[0].Title != "Updated title" {
		t.Errorf("expected Updated title, got %s", convs[0].Title)
	}

	if err := s.DeleteConversation(id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	convs, _ = s.ListConversations()
	if len(convs) != 0 {
		t.Errorf("expected 0 convs after delete, got %d", len(convs))
	}
}

func TestMessages(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	convID, _ := s.CreateConversation("Chat", "qwen2.5:14b")

	_, err := s.AddMessage(convID, "user", "Hello")
	if err != nil {
		t.Fatalf("add user msg: %v", err)
	}
	_, err = s.AddMessage(convID, "assistant", "Hi there!")
	if err != nil {
		t.Fatalf("add assistant msg: %v", err)
	}

	msgs, err := s.Messages(convID)
	if err != nil {
		t.Fatalf("messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 msgs, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("expected user, got %s", msgs[0].Role)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("expected assistant, got %s", msgs[1].Role)
	}
}

func TestDefaultTitle(t *testing.T) {
	s, cleanup := setup(t)
	defer cleanup()

	id, err := s.CreateConversation("", "qwen2.5:14b")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	convs, _ := s.ListConversations()
	if convs[0].Title != "New conversation" {
		t.Errorf("expected default title, got %s", convs[0].Title)
	}
	_ = id
}
