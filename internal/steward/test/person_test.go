package steward_test

import (
	"testing"
	"time"

	"github.com/mohitraku/sica/internal/models"
	"github.com/mohitraku/sica/internal/steward"
)

func TestValidatePersonName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "John Doe", false},
		{"single char", "A", false},
		{"128 chars", string(make([]byte, 128)), false},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"129 chars", string(make([]byte, 129)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := steward.ValidatePersonName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePersonName(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid email", "john@example.com", false},
		{"empty string", "", false},
		{"no at sign", "notanemail", true},
		{"just at sign", "@", false},
		{"at sign only", "a@b", false},
		{"255 chars email", "a@" + string(make([]byte, 253)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := steward.ValidateEmail(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestDaysSinceContact(t *testing.T) {
	today := models.Today()

	t.Run("never contacted", func(t *testing.T) {
		d := steward.DaysSinceContact("")
		if d != -1 {
			t.Errorf("expected -1 for never contacted, got %d", d)
		}
	})

	t.Run("contacted today", func(t *testing.T) {
		d := steward.DaysSinceContact(today)
		if d != 0 {
			t.Errorf("expected 0 for today, got %d", d)
		}
	})

	t.Run("contacted yesterday", func(t *testing.T) {
		yesterday := time.Now().AddDate(0, 0, -1).Format(models.DateLayout)
		d := steward.DaysSinceContact(yesterday)
		if d != 1 {
			t.Errorf("expected 1 for yesterday, got %d", d)
		}
	})

	t.Run("contacted 7 days ago", func(t *testing.T) {
		weekAgo := time.Now().AddDate(0, 0, -7).Format(models.DateLayout)
		d := steward.DaysSinceContact(weekAgo)
		if d != 7 {
			t.Errorf("expected 7 for week ago, got %d", d)
		}
	})

	t.Run("invalid date format", func(t *testing.T) {
		d := steward.DaysSinceContact("not-a-date")
		if d != -1 {
			t.Errorf("expected -1 for invalid date, got %d", d)
		}
	})

	t.Run("future date", func(t *testing.T) {
		future := time.Now().AddDate(0, 0, 1).Format(models.DateLayout)
		d := steward.DaysSinceContact(future)
		if d != 0 {
			t.Errorf("expected 0 for future date, got %d", d)
		}
	})
}
