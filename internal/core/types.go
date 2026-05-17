package core

import "time"

// Habit models a recurring habit.
type Habit struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Frequency    string    `json:"frequency"`
	TargetValue  int       `json:"target_value"`
	QuantityType string    `json:"quantity_type"`
	CreatedAt    time.Time `json:"created_at"`
}

// HabitEntry records a single completion of a habit.
type HabitEntry struct {
	ID      int64  `json:"id"`
	HabitID int64  `json:"habit_id"`
	Date    string `json:"date"`
	Value   int    `json:"value"`
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
