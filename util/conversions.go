package util

import "strconv"

func ToFloat(value string) float64 {
	_float, err := strconv.ParseFloat(value, 64)

	if err == nil {
		return _float
	}

	return 0
}
