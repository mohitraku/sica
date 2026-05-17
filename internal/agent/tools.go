package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mojitrk/sica/internal/core"
	"github.com/mojitrk/sica/internal/habits"
)

func RegisterAll(r *Registry, hStore *habits.Store) {
	registerHabitTools(r, hStore)
}

func registerHabitTools(r *Registry, s *habits.Store) {
	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "list_habits",
			Description: "List all habits with current streaks and today's progress",
			Parameters:  json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			hh, err := s.List()
			if err != nil {
				return "", err
			}
			if len(hh) == 0 {
				return "No habits found.", nil
			}
			var b strings.Builder
			for _, h := range hh {
				stats, _ := s.Stats(h.ID)
				streak := 0
				todayVal := 0
				if stats != nil {
					streak = stats.CurrentStreak
					todayVal = stats.TodayValue
				}
				fmt.Fprintf(&b, "- %s [%d/%d today, %d-day streak, id=%d]\n",
					h.Name, todayVal, h.TargetValue, streak, h.ID)
			}
			return b.String(), nil
		},
	})

	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "get_habit",
			Description: "Get detailed information about a specific habit including a longer streak history",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"id":{"type":"integer","description":"Habit ID"}},"required":["id"]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			var p struct{ ID int64 `json:"id"` }
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			h, err := s.Get(p.ID)
			if err != nil {
				return "", err
			}
			if h == nil {
				return fmt.Sprintf("Habit %d not found.", p.ID), nil
			}
			stats, _ := s.Stats(h.ID)
			streak := 0
			longest := 0
			todayVal := 0
			total := 0
			if stats != nil {
				streak = stats.CurrentStreak
				longest = stats.LongestStreak
				todayVal = stats.TodayValue
				total = stats.TotalEntries
			}
			return fmt.Sprintf("%s (id=%d)\n  Frequency: %s\n  Target: %d\n  Quantity type: %s\n  Progress today: %d/%d\n  Current streak: %d days\n  Longest streak: %d days\n  Total entries: %d",
				h.Name, h.ID, h.Frequency, h.TargetValue, h.QuantityType,
				todayVal, h.TargetValue, streak, longest, total), nil
		},
	})

	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "create_habit",
			Description: "Create a new habit. Valid frequency values: daily, weekly, monthly.",
			Parameters: json.RawMessage(`{"type":"object","properties":{
				"name":{"type":"string","description":"Name of the habit"},
				"frequency":{"type":"string","description":"daily, weekly, or monthly"},
				"target_value":{"type":"integer","description":"Target number per period, default 1"},
				"quantity_type":{"type":"string","description":"binary, count, or duration, default count"}
			},"required":["name"]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			var p struct {
				Name         string `json:"name"`
				Frequency    string `json:"frequency"`
				TargetValue  int    `json:"target_value"`
				QuantityType string `json:"quantity_type"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			if p.TargetValue <= 0 {
				p.TargetValue = 1
			}
			if p.Frequency == "" {
				p.Frequency = "daily"
			}
			h := &core.Habit{
				Name:         p.Name,
				Frequency:    p.Frequency,
				TargetValue:  p.TargetValue,
				QuantityType: p.QuantityType,
			}
			if err := s.Create(h); err != nil {
				return "", fmt.Errorf("create habit: %w", err)
			}
			return fmt.Sprintf("Created habit \"%s\" (id=%d, frequency=%s, target=%d, type=%s).", h.Name, h.ID, h.Frequency, h.TargetValue, h.QuantityType), nil
		},
	})

	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "log_habit",
			Description: "Log a value for a habit on a specific date (defaults to today). Use to record progress.",
			Parameters: json.RawMessage(`{"type":"object","properties":{
				"id":{"type":"integer","description":"Habit ID"},
				"value":{"type":"integer","description":"Value to set, default 1"},
				"date":{"type":"string","description":"Date in YYYY-MM-DD format, defaults to today"}
			},"required":["id"]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			var p struct {
				ID    int64  `json:"id"`
				Value int    `json:"value"`
				Date  string `json:"date"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			if p.Date == "" {
				p.Date = time.Now().Format("2006-01-02")
			}
			if p.Value <= 0 {
				p.Value = 1
			}
			if err := s.SetEntry(p.ID, p.Date, p.Value); err != nil {
				return "", fmt.Errorf("log habit: %w", err)
			}
			return fmt.Sprintf("Logged %d for habit %d on %s.", p.Value, p.ID, p.Date), nil
		},
	})

	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "delete_habit",
			Description: "Permanently delete a habit and all its entries",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"id":{"type":"integer","description":"Habit ID to delete"}},"required":["id"]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			var p struct{ ID int64 `json:"id"` }
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			if err := s.Delete(p.ID); err != nil {
				return "", fmt.Errorf("delete habit: %w", err)
			}
			return fmt.Sprintf("Deleted habit %d.", p.ID), nil
		},
	})

	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "update_habit",
			Description: "Update a habit's fields. Only specify the fields you want to change.",
			Parameters: json.RawMessage(`{"type":"object","properties":{
				"id":{"type":"integer","description":"Habit ID"},
				"name":{"type":"string","description":"New name"},
				"frequency":{"type":"string","description":"daily, weekly, or monthly"},
				"target_value":{"type":"integer","description":"New target per period"},
				"quantity_type":{"type":"string","description":"binary, count, or duration"}
			},"required":["id"]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			var p struct {
				ID           int64  `json:"id"`
				Name         string `json:"name"`
				Frequency    string `json:"frequency"`
				TargetValue  int    `json:"target_value"`
				QuantityType string `json:"quantity_type"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			h, err := s.Get(p.ID)
			if err != nil {
				return "", err
			}
			if h == nil {
				return fmt.Sprintf("Habit %d not found.", p.ID), nil
			}
			if p.Name != "" {
				h.Name = p.Name
			}
			if p.Frequency != "" {
				h.Frequency = p.Frequency
			}
			if p.TargetValue > 0 {
				h.TargetValue = p.TargetValue
			}
			if p.QuantityType != "" {
				h.QuantityType = p.QuantityType
			}
			if err := s.Update(h); err != nil {
				return "", fmt.Errorf("update habit: %w", err)
			}
			return fmt.Sprintf("Updated habit \"%s\" (id=%d).", h.Name, h.ID), nil
		},
	})
}
