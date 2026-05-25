package steward

import (
	"errors"
	"strings"
	"time"

	"github.com/mohitraku/sica/internal/models"
)

func ValidatePersonName(name string) error {
	if len(strings.TrimSpace(name)) == 0 {
		return errors.New("name must not be empty")
	}
	if len(name) > 128 {
		return errors.New("name must be at most 128 characters")
	}
	return nil
}

func ValidateEmail(email string) error {
	if email == "" {
		return nil
	}
	if !strings.Contains(email, "@") {
		return errors.New("email must contain @")
	}
	if len(email) > 254 {
		return errors.New("email must be at most 254 characters")
	}
	return nil
}

func DaysSinceContact(lastContacted string) int {
	if lastContacted == "" {
		return -1
	}
	t, err := time.Parse(models.DateLayout, lastContacted)
	if err != nil {
		return -1
	}
	today, _ := time.Parse(models.DateLayout, models.Today())
	diff := today.Sub(t)
	days := int(diff.Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
