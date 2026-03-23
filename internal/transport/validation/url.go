package validation

import (
	"errors"
	"strings"
)

// ErrInvalidURL means URL does not start with supported scheme.
var ErrInvalidURL = errors.New("некорректный url")

// ValidateOriginalURL validates URL scheme.
func ValidateOriginalURL(orig string) error {
	if !strings.HasPrefix(orig, "http://") && !strings.HasPrefix(orig, "https://") {
		return ErrInvalidURL
	}

	return nil
}
