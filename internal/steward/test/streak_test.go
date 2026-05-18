package steward_test

import (
	"testing"

	"github.com/mojitrk/sica/internal/models"
	"github.com/mojitrk/sica/internal/steward"
)

func makeEntry(date string, value int) models.HabitEntry {
	return models.HabitEntry{Date: date, Value: value}
}

func entries(dates ...string) []models.HabitEntry {
	var out []models.HabitEntry
	for _, d := range dates {
		out = append(out, makeEntry(d, 1))
	}
	return out
}

func TestCurrentStreakEmpty(t *testing.T) {
	h := models.Habit{Frequency: "daily", TargetValue: 1}
	got := steward.CurrentStreak(nil, h, "2026-05-18")
	if got != 0 {
		t.Errorf("expected 0 for empty entries, got %d", got)
	}
}

func TestCurrentStreakTodayDone(t *testing.T) {
	h := models.Habit{Frequency: "daily", TargetValue: 1}
	entries := entries("2026-05-15", "2026-05-16", "2026-05-17", "2026-05-18")
	got := steward.CurrentStreak(entries, h, "2026-05-18")
	if got != 4 {
		t.Errorf("expected streak 4, got %d", got)
	}
}

func TestCurrentStreakTodayUndone(t *testing.T) {
	h := models.Habit{Frequency: "daily", TargetValue: 1}
	entries := entries("2026-05-15", "2026-05-16", "2026-05-17")
	got := steward.CurrentStreak(entries, h, "2026-05-18")
	if got != 3 {
		t.Errorf("expected streak 3 (continuing from yesterday), got %d", got)
	}
}

func TestCurrentStreakGap(t *testing.T) {
	h := models.Habit{Frequency: "daily", TargetValue: 1}
	entries := entries("2026-05-10", "2026-05-11", "2026-05-15", "2026-05-16")
	got := steward.CurrentStreak(entries, h, "2026-05-18")
	if got != 0 {
		t.Errorf("expected streak 0 (gap before today), got %d", got)
	}
}

func TestCurrentStreakSingleEntry(t *testing.T) {
	h := models.Habit{Frequency: "daily", TargetValue: 1}
	got := steward.CurrentStreak(entries("2026-05-18"), h, "2026-05-18")
	if got != 1 {
		t.Errorf("expected streak 1, got %d", got)
	}
}

func TestLongestStreakConsecutive(t *testing.T) {
	h := models.Habit{Frequency: "daily", TargetValue: 1}
	entries := entries("2026-05-10", "2026-05-11", "2026-05-12", "2026-05-13", "2026-05-14")
	got := steward.LongestStreak(entries, h)
	if got != 5 {
		t.Errorf("expected longest 5, got %d", got)
	}
}

func TestLongestStreakWithGap(t *testing.T) {
	h := models.Habit{Frequency: "daily", TargetValue: 1}
	entries := entries("2026-05-10", "2026-05-11", "2026-05-12",
		"2026-05-15", "2026-05-16", "2026-05-17", "2026-05-18")
	got := steward.LongestStreak(entries, h)
	if got != 4 {
		t.Errorf("expected longest 4, got %d", got)
	}
}

func TestLongestStreakPastLonger(t *testing.T) {
	h := models.Habit{Frequency: "daily", TargetValue: 1}
	// Past streak of 7, current streak of 3
	entries := entries(
		"2026-04-01", "2026-04-02", "2026-04-03", "2026-04-04",
		"2026-04-05", "2026-04-06", "2026-04-07",
		// gap
		"2026-05-16", "2026-05-17", "2026-05-18",
	)
	got := steward.LongestStreak(entries, h)
	if got != 7 {
		t.Errorf("expected longest 7, got %d", got)
	}
}

func TestLongestStreakEmpty(t *testing.T) {
	h := models.Habit{Frequency: "daily", TargetValue: 1}
	got := steward.LongestStreak(nil, h)
	if got != 0 {
		t.Errorf("expected 0 for empty entries, got %d", got)
	}
}

func TestCurrentStreakWeekly(t *testing.T) {
	h := models.Habit{Frequency: "weekly", TargetValue: 1}
	// Monday and Wednesday of the current ISO week
	entries := []models.HabitEntry{
		makeEntry("2026-05-11", 1), // Monday
		makeEntry("2026-05-13", 1), // Wednesday (same week)
	}
	got := steward.CurrentStreak(entries, h, "2026-05-13")
	if got != 1 {
		t.Errorf("expected weekly streak 1 (one completed week), got %d", got)
	}
}

func TestCurrentStreakWeeklyMultiWeek(t *testing.T) {
	h := models.Habit{Frequency: "weekly", TargetValue: 1}
	entries := []models.HabitEntry{
		makeEntry("2026-04-27", 1), // Week 18
		makeEntry("2026-05-04", 1), // Week 19
		makeEntry("2026-05-11", 1), // Week 20 (current)
	}
	got := steward.CurrentStreak(entries, h, "2026-05-12")
	if got != 3 {
		t.Errorf("expected weekly streak 3, got %d", got)
	}
}

func TestCurrentStreakWeeklyGap(t *testing.T) {
	h := models.Habit{Frequency: "weekly", TargetValue: 1}
	entries := []models.HabitEntry{
		makeEntry("2026-04-27", 1), // Week 18
		// Week 19 missing
		makeEntry("2026-05-11", 1), // Week 20 (current)
	}
	got := steward.CurrentStreak(entries, h, "2026-05-12")
	if got != 1 {
		t.Errorf("expected weekly streak 1 (gap at week 19), got %d", got)
	}
}

func TestCurrentStreakMonthly(t *testing.T) {
	h := models.Habit{Frequency: "monthly", TargetValue: 1}
	entries := []models.HabitEntry{
		makeEntry("2026-03-15", 1), // March
		makeEntry("2026-04-10", 1), // April
		makeEntry("2026-05-01", 1), // May (current)
	}
	got := steward.CurrentStreak(entries, h, "2026-05-18")
	if got != 3 {
		t.Errorf("expected monthly streak 3, got %d", got)
	}
}

func TestCurrentStreakMonthlyGap(t *testing.T) {
	h := models.Habit{Frequency: "monthly", TargetValue: 1}
	entries := []models.HabitEntry{
		makeEntry("2026-02-10", 1), // Feb
		// March missing
		makeEntry("2026-04-05", 1), // April
		makeEntry("2026-05-10", 1), // May
	}
	got := steward.CurrentStreak(entries, h, "2026-05-18")
	if got != 2 {
		t.Errorf("expected monthly streak 2, got %d", got)
	}
}

func TestWeeklyStreakAcrossISOYearBoundary(t *testing.T) {
	// 2025-W52: Mon Dec 22 through Sun Dec 28, 2025
	// 2026-W01: Mon Dec 29, 2025 through Sun Jan 4, 2026
	// These are consecutive ISO weeks even though the calendar year changes.
	h := models.Habit{Frequency: "weekly", TargetValue: 1}
	entries := []models.HabitEntry{
		makeEntry("2025-12-22", 1), // ISO week 2025-W52
		makeEntry("2025-12-29", 1), // ISO week 2026-W01
	}
	got := steward.CurrentStreak(entries, h, "2025-12-30")
	if got != 2 {
		t.Errorf("expected weekly streak 2 across ISO year boundary, got %d", got)
	}
}

func TestWeeklyStreakBackwardAcrossISOYearBoundary(t *testing.T) {
	h := models.Habit{Frequency: "weekly", TargetValue: 1}
	entries := []models.HabitEntry{
		makeEntry("2026-01-05", 1), // ISO week 2026-W02 (Mon Jan 5, 2026)
		makeEntry("2025-12-29", 1), // ISO week 2026-W01
		makeEntry("2025-12-22", 1), // ISO week 2025-W52
	}
	// Longest streak should be 3, spanning 2025-W52, 2026-W01, 2026-W02
	got := steward.LongestStreak(entries, h)
	if got != 3 {
		t.Errorf("expected longest weekly streak 3 across ISO boundary, got %d", got)
	}
}

func TestLongestStreakAtAllTimeBest(t *testing.T) {
	h := models.Habit{Frequency: "daily", TargetValue: 1}
	// The only streak is the current one, so longest == current
	entries := entries("2026-05-16", "2026-05-17", "2026-05-18")
	got := steward.LongestStreak(entries, h)
	if got != 3 {
		t.Errorf("expected longest 3, got %d", got)
	}
}
