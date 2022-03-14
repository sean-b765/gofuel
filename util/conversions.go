package util

import "strconv"

func ToFloat(value string) float64 {
	_float, err := strconv.ParseFloat(value, 64)

	if err == nil {
		return _float
	}

	return 0
}

func FromFloat(value float64) string {
	_string := strconv.FormatFloat(value, 'f', -1, 32)
	return _string
}
