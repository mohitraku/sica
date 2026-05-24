package steward

import (
	"errors"
	"strings"
)

func ValidateTaskTitle(title string) error {
	if len(strings.TrimSpace(title)) == 0 {
		return errors.New("task title must not be empty")
	}
	if len(title) > 128 {
		return errors.New("task title must be at most 128 characters")
	}
	return nil
}
