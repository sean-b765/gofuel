package util

import "time"

func FormatDate(layout, value string) string {
	t, err := time.Parse(layout, value)
	if err != nil {
		return value
	}
	return t.Format("2006-01-02")
}

var dateLayouts = []string{
	"02/01/2006 15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	time.RFC3339,
	"02/01/2006 15:04",
	"02/01/2006",
	"2006-01-02",
}

func NormaliseDate(value string) string {
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, value); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return value
}