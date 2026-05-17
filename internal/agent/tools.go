package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mojitrk/sica/internal/core"
	"github.com/mojitrk/sica/internal/habits"
	"github.com/mojitrk/sica/internal/tasks"
)

func RegisterAll(r *Registry, hStore *habits.Store, tStore *tasks.Store) {
	registerHabitTools(r, hStore)
	registerTaskTools(r, tStore)
}

func registerHabitTools(r *Registry, s *habits.Store) {
	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "list_habits",
			Description: "List all habits with current streaks and today's progress",
			Parameters:  json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			hh, err := s.List(false)
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
			return fmt.Sprintf("%s (id=%d)\n  Description: %s\n  Frequency: %s\n  Target: %d\n  Progress today: %d/%d\n  Current streak: %d days\n  Longest streak: %d days\n  Total entries: %d",
				h.Name, h.ID, h.Description, h.Frequency, h.TargetValue,
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
				"description":{"type":"string","description":"Optional description"}
			},"required":["name"]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			var p struct {
				Name        string `json:"name"`
				Frequency   string `json:"frequency"`
				TargetValue int    `json:"target_value"`
				Description string `json:"description"`
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
				Name:        p.Name,
				Frequency:   p.Frequency,
				TargetValue: p.TargetValue,
				Description: p.Description,
			}
			if err := s.Create(h); err != nil {
				return "", fmt.Errorf("create habit: %w", err)
			}
			return fmt.Sprintf("Created habit \"%s\" (id=%d, frequency=%s, target=%d).", h.Name, h.ID, h.Frequency, h.TargetValue), nil
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
			if err := s.SetEntry(p.ID, p.Date, p.Value, ""); err != nil {
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
}

func registerTaskTools(r *Registry, s *tasks.Store) {
	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "list_tasks",
			Description: "List tasks with optional filters for status and priority",
			Parameters: json.RawMessage(`{"type":"object","properties":{
				"status":{"type":"string","description":"Filter: todo, done, or empty for all"},
				"priority":{"type":"string","description":"Filter: high, med, low, or empty for all"},
				"project_id":{"type":"integer","description":"Filter by project ID, optional"}
			},"required":[]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			var p struct {
				Status    string `json:"status"`
				Priority  string `json:"priority"`
				ProjectID *int64 `json:"project_id"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			f := tasks.Filter{
				Status:    p.Status,
				Priority:  p.Priority,
				ProjectID: p.ProjectID,
			}
			tt, err := s.ListTasks(f)
			if err != nil {
				return "", err
			}
			if len(tt) == 0 {
				return "No tasks found matching the filters.", nil
			}
			var b strings.Builder
			for _, t := range tt {
				due := ""
				if t.DueDate != nil {
					due = fmt.Sprintf(" due=%s", t.DueDate.Format("2006-01-02"))
				}
				fmt.Fprintf(&b, "- [%s] %s (%s%s, id=%d)\n", t.Status, t.Title, t.Priority, due, t.ID)
			}
			return b.String(), nil
		},
	})

	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "get_task",
			Description: "Get full details of a specific task",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"id":{"type":"integer","description":"Task ID"}},"required":["id"]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			var p struct{ ID int64 `json:"id"` }
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			t, err := s.GetTask(p.ID)
			if err != nil {
				return "", err
			}
			if t == nil {
				return fmt.Sprintf("Task %d not found.", p.ID), nil
			}
			due := "none"
			if t.DueDate != nil {
				due = t.DueDate.Format("2006-01-02")
			}
			done := ""
			if t.CompletedAt != nil {
				done = fmt.Sprintf("\n  Completed: %s", t.CompletedAt.Format("2006-01-02"))
			}
			proj := "none"
			if t.ProjectID != nil {
				proj = fmt.Sprintf("%d", *t.ProjectID)
			}
			return fmt.Sprintf("%s (id=%d)\n  Status: %s\n  Priority: %s\n  Due date: %s\n  Project ID: %s\n  Description: %s%s",
				t.Title, t.ID, t.Status, t.Priority, due, proj, t.Description, done), nil
		},
	})

	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "create_task",
			Description: "Create a new task. Priority defaults to med, status defaults to todo.",
			Parameters: json.RawMessage(`{"type":"object","properties":{
				"title":{"type":"string","description":"Task title"},
				"priority":{"type":"string","description":"high, med, or low"},
				"due_date":{"type":"string","description":"Due date in YYYY-MM-DD format, optional"},
				"description":{"type":"string","description":"Optional description"},
				"project_id":{"type":"integer","description":"Optional project ID"}
			},"required":["title"]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			var p struct {
				Title       string `json:"title"`
				Priority    string `json:"priority"`
				DueDate     string `json:"due_date"`
				Description string `json:"description"`
				ProjectID   *int64 `json:"project_id"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			if p.Priority == "" {
				p.Priority = "med"
			}
			t := &core.Task{
				Title:       p.Title,
				Priority:    p.Priority,
				Description: p.Description,
				Status:      "todo",
				ProjectID:   p.ProjectID,
			}
			if p.DueDate != "" {
				d, err := time.Parse("2006-01-02", p.DueDate)
				if err != nil {
					return "", fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
				}
				t.DueDate = &d
			}
			if err := s.CreateTask(t); err != nil {
				return "", fmt.Errorf("create task: %w", err)
			}
			dueStr := ""
			if t.DueDate != nil {
				dueStr = fmt.Sprintf(", due=%s", t.DueDate.Format("2006-01-02"))
			}
			return fmt.Sprintf("Created task \"%s\" (id=%d, priority=%s%s).", t.Title, t.ID, t.Priority, dueStr), nil
		},
	})

	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "complete_task",
			Description: "Mark a task as done",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"id":{"type":"integer","description":"Task ID to complete"}},"required":["id"]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			var p struct{ ID int64 `json:"id"` }
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			if err := s.CompleteTask(p.ID); err != nil {
				return "", fmt.Errorf("complete task: %w", err)
			}
			return fmt.Sprintf("Task %d marked as done.", p.ID), nil
		},
	})

	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "update_task",
			Description: "Update task fields. Only specify the fields you want to change.",
			Parameters: json.RawMessage(`{"type":"object","properties":{
				"id":{"type":"integer","description":"Task ID"},
				"title":{"type":"string","description":"New title"},
				"priority":{"type":"string","description":"high, med, or low"},
				"due_date":{"type":"string","description":"Due date in YYYY-MM-DD format"},
				"description":{"type":"string","description":"New description"},
				"project_id":{"type":"integer","description":"New project ID"}
			},"required":["id"]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			var p struct {
				ID          int64  `json:"id"`
				Title       string `json:"title"`
				Priority    string `json:"priority"`
				DueDate     string `json:"due_date"`
				Description string `json:"description"`
				ProjectID   *int64 `json:"project_id"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			t, err := s.GetTask(p.ID)
			if err != nil {
				return "", fmt.Errorf("get task: %w", err)
			}
			if t == nil {
				return fmt.Sprintf("Task %d not found.", p.ID), nil
			}
			if p.Title != "" {
				t.Title = p.Title
			}
			if p.Priority != "" {
				t.Priority = p.Priority
			}
			if p.Description != "" {
				t.Description = p.Description
			}
			if p.DueDate != "" {
				d, err := time.Parse("2006-01-02", p.DueDate)
				if err != nil {
					return "", fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
				}
				t.DueDate = &d
			}
			if p.ProjectID != nil {
				t.ProjectID = p.ProjectID
			}
			if err := s.UpdateTask(t); err != nil {
				return "", fmt.Errorf("update task: %w", err)
			}
			return fmt.Sprintf("Updated task %d.", t.ID), nil
		},
	})

	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "delete_task",
			Description: "Permanently delete a task",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"id":{"type":"integer","description":"Task ID to delete"}},"required":["id"]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			var p struct{ ID int64 `json:"id"` }
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			if err := s.DeleteTask(p.ID); err != nil {
				return "", fmt.Errorf("delete task: %w", err)
			}
			return fmt.Sprintf("Deleted task %d.", p.ID), nil
		},
	})

	r.Register(Tool{
		Definition: ToolDefinition{
			Name:        "list_projects",
			Description: "List all task projects",
			Parameters:  json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
		},
		Fn: func(args json.RawMessage) (string, error) {
			pp, err := s.ListProjects()
			if err != nil {
				return "", err
			}
			if len(pp) == 0 {
				return "No projects found.", nil
			}
			var b strings.Builder
			for _, p := range pp {
				fmt.Fprintf(&b, "- %s (id=%d)\n", p.Name, p.ID)
			}
			return b.String(), nil
		},
	})
}
