package steward

import (
	"fmt"
	"slices"
	"time"

	"github.com/mohitraku/sica/internal/models"
)

func CurrentStreak(entries []models.RoutineEntry, r models.Routine, today string) int {
	if len(entries) == 0 {
		return 0
	}
	return countBackward(entries, r, today)
}

func LongestStreak(entries []models.RoutineEntry, r models.Routine) int {
	if len(entries) == 0 {
		return 0
	}

	periods := completedPeriods(entries, r)
	if len(periods) == 0 {
		return 0
	}

	periodList := sortedKeys(periods)

	longest := 0
	current := 1
	for i := 1; i < len(periodList); i++ {
		if isConsecutive(periodList[i-1], periodList[i], r.Frequency) {
			current++
		} else {
			if current > longest {
				longest = current
			}
			current = 1
		}
	}
	if current > longest {
		longest = current
	}
	return longest
}

func countBackward(entries []models.RoutineEntry, r models.Routine, today string) int {
	periods := completedPeriods(entries, r)
	cur := periodKey(today, r.Frequency)

	if !periods[cur] {
		cur = prevPeriod(cur, r.Frequency)
	}

	streak := 0
	for cur != "" && periods[cur] {
		streak++
		cur = prevPeriod(cur, r.Frequency)
	}
	return streak
}

func completedPeriods(entries []models.RoutineEntry, r models.Routine) map[string]bool {
	periods := make(map[string]bool)
	for _, e := range entries {
		if IsDone(e.Value, r.TargetValue) {
			periods[periodKey(e.Date, r.Frequency)] = true
		}
	}
	return periods
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// periodKey converts a date string to a period identifier based on frequency.
// daily: "2026-05-18", weekly: "2026-W20", monthly: "2026-05"
func periodKey(dateStr, frequency string) string {
	t, err := time.Parse(models.DateLayout, dateStr)
	if err != nil {
		return dateStr
	}
	switch frequency {
	case "daily":
		return dateStr
	case "weekly":
		year, week := t.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", year, week)
	case "monthly":
		return t.Format("2006-01")
	}
	return dateStr
}

func prevPeriod(period, frequency string) string {
	switch frequency {
	case "daily":
		t, err := time.Parse("2006-01-02", period)
		if err != nil {
			return ""
		}
		return t.AddDate(0, 0, -1).Format("2006-01-02")
	case "weekly":
		return shiftWeek(period, -1)
	case "monthly":
		t, err := time.Parse("2006-01", period)
		if err != nil {
			return ""
		}
		return t.AddDate(0, -1, 0).Format("2006-01")
	}
	return ""
}

func isConsecutive(a, b, frequency string) bool {
	expected := nextPeriod(a, frequency)
	return expected == b
}

func nextPeriod(period, frequency string) string {
	switch frequency {
	case "daily":
		t, err := time.Parse("2006-01-02", period)
		if err != nil {
			return ""
		}
		return t.AddDate(0, 0, 1).Format("2006-01-02")
	case "weekly":
		return shiftWeek(period, +1)
	case "monthly":
		t, err := time.Parse("2006-01", period)
		if err != nil {
			return ""
		}
		return t.AddDate(0, 1, 0).Format("2006-01")
	}
	return ""
}

func shiftWeek(period string, delta int) string {
	var year, week int
	if _, err := fmt.Sscanf(period, "%d-W%d", &year, &week); err != nil {
		return ""
	}
	// Jan 4 is always in ISO week 1 — find the Monday of that week.
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	daysToMonday := (jan4.Weekday() - time.Monday + 7) % 7
	mon := jan4.AddDate(0, 0, -int(daysToMonday))
	// Advance to the Monday of the target week, then shift.
	mon = mon.AddDate(0, 0, (week-1)*7+delta*7)
	y, w := mon.ISOWeek()
	return fmt.Sprintf("%04d-W%02d", y, w)
}
