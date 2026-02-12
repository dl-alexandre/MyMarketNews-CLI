package util

import (
	"strings"
	"time"
)

var centralLocation = mustLoadLocation("America/Chicago")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("CST", -6*3600)
	}
	return loc
}

func ParseDate(value string) (time.Time, bool) {
	t, _, ok := ParseDateWithLayout(value)
	return t, ok
}

func ParseDateWithLayout(value string) (time.Time, string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, "", false
	}

	layouts := []string{
		"01/02/2006",
		"2006-01-02",
		"01/02/2006 15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, centralLocation); err == nil {
			return t, layout, true
		}
	}

	return time.Time{}, "", false
}

func DaysBetween(start, end time.Time) int {
	if end.Before(start) {
		start, end = end, start
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
	return int(end.Sub(start).Hours()/24) + 1
}
