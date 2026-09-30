package timeutil

import "time"

// FormatUTC is a shortcut of t.UTC().Format() to set the timezone to UTC before formatting.
func FormatUTC(t time.Time, format string) string {
	return t.UTC().Format(format)
}
