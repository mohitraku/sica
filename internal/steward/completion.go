package steward

import "github.com/mohitraku/sica/internal/models"

func IsDone(value, target int) bool {
	return value >= target
}

func PeriodValue(entries []models.RoutineEntry, r models.Routine, date string) int {
	key := periodKey(date, r.Frequency)
	total := 0
	for _, e := range entries {
		if periodKey(e.Date, r.Frequency) == key {
			total += e.Value
		}
	}
	return total
}

func IsPeriodDone(entries []models.RoutineEntry, r models.Routine, date string) bool {
	return PeriodValue(entries, r, date) >= r.TargetValue
}

func IncrementValue(current, target int) int {
	if current >= target {
		return current
	}
	return current + 1
}

func DecrementValue(current int) int {
	if current <= 0 {
		return 0
	}
	return current - 1
}
