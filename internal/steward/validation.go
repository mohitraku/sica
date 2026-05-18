package steward

import (
	"errors"
	"strings"

	"github.com/mojitrk/sica/internal/models"
)

var validFrequencies = map[string]bool{"daily": true, "weekly": true, "monthly": true}
var validQuantityTypes = map[string]bool{"binary": true, "count": true}

func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name must not be empty")
	}
	if len(name) > 128 {
		return errors.New("name must be 128 characters or fewer")
	}
	return nil
}

func ValidateFrequency(f string) bool {
	return validFrequencies[f]
}

func ValidateQuantityType(q string) bool {
	return validQuantityTypes[q]
}

func NormalizeHabit(h *models.Habit) {
	if !ValidateFrequency(h.Frequency) {
		h.Frequency = "daily"
	}
	if !ValidateQuantityType(h.QuantityType) {
		h.QuantityType = "binary"
	}
	if h.QuantityType == "binary" {
		h.TargetValue = 1
	}
	if h.TargetValue < 1 {
		h.TargetValue = 1
	}
}
