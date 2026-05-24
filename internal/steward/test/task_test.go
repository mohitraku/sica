package steward_test

import (
	"testing"

	"github.com/mohitraku/sica/internal/steward"
)

func TestValidateTaskTitle(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid title", "Fix the sink", false},
		{"single char", "A", false},
		{"128 chars", string(make([]byte, 128)), false},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"129 chars", string(make([]byte, 129)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := steward.ValidateTaskTitle(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTaskTitle(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
