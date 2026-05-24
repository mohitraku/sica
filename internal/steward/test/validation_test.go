package steward_test

import (
	"testing"

	"github.com/mohitraku/sica/internal/models"
	"github.com/mohitraku/sica/internal/steward"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "Drink water", false},
		{"single char", "A", false},
		{"128 chars", string(make([]byte, 128)), false},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"129 chars", string(make([]byte, 129)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := steward.ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateFrequency(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"daily", true},
		{"weekly", true},
		{"monthly", true},
		{"yearly", false},
		{"", false},
		{"DAILY", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := steward.ValidateFrequency(tt.input)
			if got != tt.want {
				t.Errorf("ValidateFrequency(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateQuantityType(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"binary", true},
		{"count", true},
		{"", false},
		{"boolean", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := steward.ValidateQuantityType(tt.input)
			if got != tt.want {
				t.Errorf("ValidateQuantityType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeRoutine(t *testing.T) {
	tests := []struct {
		name     string
		input    models.Routine
		wantFreq string
		wantQty  string
		wantTarg int
	}{
		{
			name:     "valid binary daily",
			input:    models.Routine{Frequency: "daily", QuantityType: "binary", TargetValue: 5},
			wantFreq: "daily",
			wantQty:  "binary",
			wantTarg: 1, // binary forces target to 1
		},
		{
			name:     "valid count monthly",
			input:    models.Routine{Frequency: "monthly", QuantityType: "count", TargetValue: 8},
			wantFreq: "monthly",
			wantQty:  "count",
			wantTarg: 8,
		},
		{
			name:     "invalid frequency defaults to daily",
			input:    models.Routine{Frequency: "yearly", QuantityType: "binary", TargetValue: 1},
			wantFreq: "daily",
			wantQty:  "binary",
			wantTarg: 1,
		},
		{
			name:     "invalid quantity type defaults to binary",
			input:    models.Routine{Frequency: "weekly", QuantityType: "boolean", TargetValue: 10},
			wantFreq: "weekly",
			wantQty:  "binary",
			wantTarg: 1, // binary forces 1
		},
		{
			name:     "zero target value clamped to 1",
			input:    models.Routine{Frequency: "daily", QuantityType: "count", TargetValue: 0},
			wantFreq: "daily",
			wantQty:  "count",
			wantTarg: 1,
		},
		{
			name:     "negative target value clamped to 1",
			input:    models.Routine{Frequency: "daily", QuantityType: "count", TargetValue: -5},
			wantFreq: "daily",
			wantQty:  "count",
			wantTarg: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &tt.input
			steward.NormalizeRoutine(r)
			if r.Frequency != tt.wantFreq {
				t.Errorf("frequency = %q, want %q", r.Frequency, tt.wantFreq)
			}
			if r.QuantityType != tt.wantQty {
				t.Errorf("quantity_type = %q, want %q", r.QuantityType, tt.wantQty)
			}
			if r.TargetValue != tt.wantTarg {
				t.Errorf("target_value = %d, want %d", r.TargetValue, tt.wantTarg)
			}
		})
	}
}
