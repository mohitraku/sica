package steward

import (
	"errors"
	"strings"

	"github.com/mohitraku/sica/internal/models"
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

func NormalizeRoutine(r *models.Routine) {
	if !ValidateFrequency(r.Frequency) {
		r.Frequency = "daily"
	}
	if !ValidateQuantityType(r.QuantityType) {
		r.QuantityType = "binary"
	}
	if r.QuantityType == "binary" {
		r.TargetValue = 1
	}
	if r.TargetValue < 1 {
		r.TargetValue = 1
	}
}
