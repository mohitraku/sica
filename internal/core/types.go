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
