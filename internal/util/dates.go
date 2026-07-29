package util

import "time"

func FormatDate(layout, value string) string {
	t, err := time.Parse(layout, value)
	if err != nil {
		return value
	}
	return t.Format("2006-01-02")
}