package steward_test

import (
	"testing"

	"github.com/mojitrk/sica/internal/steward"
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
