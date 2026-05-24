package steward_test

import (
	"testing"

	"github.com/mohitraku/sica/internal/models"
	"github.com/mohitraku/sica/internal/steward"
)

func TestIsDone(t *testing.T) {
	tests := []struct {
		name   string
		value  int
		target int
		want   bool
	}{
		{"done exactly at target", 5, 5, true},
		{"done above target", 8, 5, true},
		{"undone below target", 3, 5, false},
		{"undone zero", 0, 1, false},
		{"done both zero", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := steward.IsDone(tt.value, tt.target)
			if got != tt.want {
				t.Errorf("IsDone(%d, %d) = %v, want %v", tt.value, tt.target, got, tt.want)
			}
		})
	}
}

func TestIncrementValue(t *testing.T) {
	tests := []struct {
		name    string
		current int
		target  int
		want    int
	}{
		{"zero to one", 0, 5, 1},
		{"partial increment", 3, 5, 4},
		{"at target stays", 5, 5, 5},
		{"above target stays", 8, 5, 8},
		{"zero to one binary", 0, 1, 1},
		{"at target binary stays", 1, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := steward.IncrementValue(tt.current, tt.target)
			if got != tt.want {
				t.Errorf("IncrementValue(%d, %d) = %d, want %d", tt.current, tt.target, got, tt.want)
			}
		})
	}
}

func TestPeriodValue(t *testing.T) {
	// 2026-05-18 (Mon) and 2026-05-22 (Fri) are in the same ISO week.
	// 2026-05-01 and 2026-05-15 are in the same month (May).
	tests := []struct {
		name      string
		frequency string
		target    int
		entries   []models.RoutineEntry
		date      string
		wantVal   int
	}{
		{
			name:      "daily exact match",
			frequency: "daily",
			target:    1,
			entries:   []models.RoutineEntry{{Date: "2026-05-18", Value: 1}},
			date:      "2026-05-18",
			wantVal:   1,
		},
		{
			name:      "daily no match",
			frequency: "daily",
			target:    1,
			entries:   []models.RoutineEntry{{Date: "2026-05-17", Value: 1}},
			date:      "2026-05-18",
			wantVal:   0,
		},
		{
			name:      "weekly sums same week",
			frequency: "weekly",
			target:    1,
			entries:   []models.RoutineEntry{
				{Date: "2026-05-18", Value: 1},
				{Date: "2026-05-22", Value: 1},
			},
			date:    "2026-05-20",
			wantVal: 2,
		},
		{
			name:      "weekly other week returns 0",
			frequency: "weekly",
			target:    1,
			entries:   []models.RoutineEntry{{Date: "2026-05-11", Value: 1}},
			date:      "2026-05-18",
			wantVal:   0,
		},
		{
			name:      "weekly partial below target",
			frequency: "weekly",
			target:    5,
			entries:   []models.RoutineEntry{{Date: "2026-05-18", Value: 2}},
			date:      "2026-05-20",
			wantVal:   2,
		},
		{
			name:      "monthly sums same month",
			frequency: "monthly",
			target:    1,
			entries:   []models.RoutineEntry{
				{Date: "2026-05-01", Value: 1},
				{Date: "2026-05-15", Value: 1},
			},
			date:    "2026-05-20",
			wantVal: 2,
		},
		{
			name:      "monthly other month returns 0",
			frequency: "monthly",
			target:    1,
			entries:   []models.RoutineEntry{{Date: "2026-04-15", Value: 1}},
			date:      "2026-05-15",
			wantVal:   0,
		},
		{
			name:      "empty entries",
			frequency: "weekly",
			target:    1,
			entries:   []models.RoutineEntry{},
			date:      "2026-05-18",
			wantVal:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := models.Routine{Frequency: tt.frequency, TargetValue: tt.target}
			got := steward.PeriodValue(tt.entries, r, tt.date)
			if got != tt.wantVal {
				t.Errorf("PeriodValue() = %d, want %d", got, tt.wantVal)
			}
		})
	}
}

func TestIsPeriodDone(t *testing.T) {
	tests := []struct {
		name      string
		frequency string
		target    int
		entries   []models.RoutineEntry
		date      string
		want      bool
	}{
		{
			name:      "daily done",
			frequency: "daily",
			target:    1,
			entries:   []models.RoutineEntry{{Date: "2026-05-18", Value: 1}},
			date:      "2026-05-18",
			want:      true,
		},
		{
			name:      "daily not done",
			frequency: "daily",
			target:    1,
			entries:   []models.RoutineEntry{},
			date:      "2026-05-18",
			want:      false,
		},
		{
			name:      "weekly done across multiple days",
			frequency: "weekly",
			target:    3,
			entries:   []models.RoutineEntry{
				{Date: "2026-05-18", Value: 2},
				{Date: "2026-05-22", Value: 1},
			},
			date: "2026-05-20",
			want: true,
		},
		{
			name:      "monthly not done below target",
			frequency: "monthly",
			target:    5,
			entries:   []models.RoutineEntry{
				{Date: "2026-05-01", Value: 1},
				{Date: "2026-05-15", Value: 2},
			},
			date: "2026-05-20",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := models.Routine{Frequency: tt.frequency, TargetValue: tt.target}
			got := steward.IsPeriodDone(tt.entries, r, tt.date)
			if got != tt.want {
				t.Errorf("IsPeriodDone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecrementValue(t *testing.T) {
	tests := []struct {
		name    string
		current int
		want    int
	}{
		{"one to zero", 1, 0},
		{"zero stays zero", 0, 0},
		{"partial decrement", 5, 4},
		{"two to one", 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := steward.DecrementValue(tt.current)
			if got != tt.want {
				t.Errorf("DecrementValue(%d) = %d, want %d", tt.current, got, tt.want)
			}
		})
	}
}
