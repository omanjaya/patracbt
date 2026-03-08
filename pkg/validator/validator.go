package validator

import (
	"errors"
	"strings"
)

// Required checks that a string field is non-empty.
func Required(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(field + " wajib diisi")
	}
	return nil
}

// MinLen checks minimum string length.
func MinLen(field, value string, min int) error {
	if len(strings.TrimSpace(value)) < min {
		return errors.New(field + " minimal " + string(rune('0'+min)) + " karakter")
	}
	return nil
}

// Validate runs multiple validators and returns combined errors.
func Validate(errs ...error) error {
	var msgs []string
	for _, e := range errs {
		if e != nil {
			msgs = append(msgs, e.Error())
		}
	}
	if len(msgs) > 0 {
		return errors.New(strings.Join(msgs, "; "))
	}
	return nil
}
