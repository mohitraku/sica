package models

import "time"

type Habit struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Frequency    string `json:"frequency"`
	TargetValue  int    `json:"target_value"`
	QuantityType string `json:"quantity_type"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type HabitEntry struct {
	ID       int64  `json:"id"`
	HabitID  string `json:"habit_id"`
	Date     string `json:"date"`
	Value    int    `json:"value"`
	LoggedAt string `json:"logged_at"`
}

func NowUTC() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

const DateLayout = "2006-01-02"

func Today() string {
	return time.Now().Format(DateLayout)
}

func IsToday(date string) bool {
	return date == Today()
}
