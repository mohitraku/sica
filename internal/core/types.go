package core

import "time"

// Habit models a recurring habit.
type Habit struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Frequency   string     `json:"frequency"`
	TargetValue int        `json:"target_value"`
	Color       string     `json:"color,omitempty"`
	Icon        string     `json:"icon,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
}

// HabitEntry records a single completion of a habit.
type HabitEntry struct {
	ID      int64     `json:"id"`
	HabitID int64     `json:"habit_id"`
	Date    string    `json:"date"`
	Value   int       `json:"value"`
	Notes   string    `json:"notes,omitempty"`
}

// Task models a to-do item.
type Task struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	ProjectID   *int64     `json:"project_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// Project groups tasks.
type Project struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Color       string    `json:"color,omitempty"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
}

// Transaction is a financial transaction.
type Transaction struct {
	ID          int64     `json:"id"`
	Amount      int64     `json:"amount"`
	Type        string    `json:"type"`
	Category    string    `json:"category"`
	Description string    `json:"description,omitempty"`
	Date        string    `json:"date"`
	CreatedAt   time.Time `json:"created_at"`
}

// Budget tracks spending limits per category.
type Budget struct {
	ID          int64     `json:"id"`
	Category    string    `json:"category"`
	AmountCents int64     `json:"amount_cents"`
	Period      string    `json:"period"`
	StartDate   string    `json:"start_date"`
}

// CalendarEvent is a calendar entry, local or synced from Outlook.
type CalendarEvent struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	Location     string    `json:"location,omitempty"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Source       string    `json:"source"`
	OutlookID    string    `json:"outlook_id,omitempty"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
}

// KnowledgeDoc is metadata for a stored document.
type KnowledgeDoc struct {
	ID        int64     `json:"id"`
	FilePath  string    `json:"file_path"`
	Title     string    `json:"title"`
	SourceURL string    `json:"source_url,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Conversation is an AI chat session.
type Conversation struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
}

// Message is a single message in an AI conversation.
type Message struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	ToolCalls      string    `json:"tool_calls,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
