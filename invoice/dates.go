package invoice

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalidDate is returned by ParseDate for an unsupported value.
var ErrInvalidDate = errors.New("invalid date")

// DateLayouts are the accepted date formats, tried in order.
var DateLayouts = []string{"02/01/2006", "2006-01-02", "20060102"}

// ParseDate parses "DD/MM/YYYY", "YYYY-MM-DD" or "YYYYMMDD".
func ParseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range DateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: %q", ErrInvalidDate, s)
}
