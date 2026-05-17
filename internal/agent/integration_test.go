package agent

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/mojitrk/sica/internal/chat"
	"github.com/mojitrk/sica/internal/core"
	"github.com/mojitrk/sica/internal/habits"
)

func TestAgentEndToEnd(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test; set INTEGRATION=1 to run")
	}

	dir, err := os.MkdirTemp("", "sica-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := sql.Open("sqlite", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS habits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL, description TEXT DEFAULT '',
			frequency TEXT NOT NULL DEFAULT 'daily',
			target_value INTEGER NOT NULL DEFAULT 1,
			quantity_type TEXT NOT NULL DEFAULT 'count',
			color TEXT DEFAULT '', icon TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			archived_at TEXT
		);
		CREATE TABLE IF NOT EXISTS habit_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			habit_id INTEGER NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
			date TEXT NOT NULL, value INTEGER NOT NULL DEFAULT 1,
			notes TEXT DEFAULT '', UNIQUE(habit_id, date)
		);
		CREATE TABLE IF NOT EXISTS ai_conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL DEFAULT 'New conversation',
			model TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS ai_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id INTEGER NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
			role TEXT NOT NULL, content TEXT NOT NULL DEFAULT '',
			tool_calls TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	hStore := habits.NewStore(db)
	chStore := chat.NewStore(db)

	err = hStore.Create(&core.Habit{
		Name: "brush teeth", Frequency: "daily", TargetValue: 1, QuantityType: "binary",
	})
	if err != nil {
		t.Fatal(err)
	}

	convID, err := chStore.CreateConversation("test", "llama3.1:8b")
	if err != nil {
		t.Fatal(err)
	}

	backend := chat.NewClient("http://localhost:11434", "llama3.1:8b")
	reg := NewRegistry()
	RegisterAll(reg, hStore)

	agt := New(backend, chStore, reg, convID, "llama3.1:8b")

	_, err = chStore.AddMessage(convID, "user", "use the list_habits tool to show me all habits")
	if err != nil {
		t.Fatal(err)
	}

	// Log what messages are being built
	chatMsgs, _ := agt.buildMessages()
	for i, m := range chatMsgs {
		t.Logf("msg[%d] role=%s content=%s tools=%d", i, m.Role, truncate(m.Content, 80), len(m.ToolCalls))
	}

	tools := reg.Definitions()
	t.Logf("tools count: %d", len(tools))

	// Dump the full JSON payload that gets sent to Ollama
	reqJSON, _ := json.Marshal(map[string]any{
		"model":    "llama3.1:8b",
		"messages": chatMsgs,
		"stream":   true,
		"tools":    tools,
		"options":  map[string]any{"num_ctx": 16384},
	})
	t.Logf("Request payload size: %d bytes", len(reqJSON))
	if len(reqJSON) > 16000 {
		t.Logf("WARNING: payload may exceed context window")
	}

	result, err := agt.RunStream(Callbacks{})
	if err != nil {
		t.Fatalf("RunStream error: %v", err)
	}
	t.Logf("Agent response: %s", result)

	if result == "No response from model — it may not support tool calling. Try a model with native tool support (llama3.2, mistral, etc.)." {
		t.Error("Agent returned the 'no tool support' fallback message")
	}

	dbMsgs, err := chStore.Messages(convID)
	if err != nil {
		t.Fatal(err)
	}

	hasToolResult := false
	for _, m := range dbMsgs {
		if m.Role == "tool" {
			hasToolResult = true
			t.Logf("Tool result: %s", m.Content)
			break
		}
	}
	if !hasToolResult {
		for _, m := range dbMsgs {
			t.Logf("DB msg: role=%s content=%s tool_calls=%s", m.Role, truncate(m.Content, 60), m.ToolCalls)
		}
		t.Error("Agent did NOT call any tools — tool calling is broken")
	} else {
		t.Log("Tool calling works end-to-end")
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
