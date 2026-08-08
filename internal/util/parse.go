package util

import (
	"errors"
	"strings"
)

/*
 * Parse coordinates URL param
 */
func ParseCoordinates(coordinates string) ([2]float64, error) {
	coords := strings.Split(coordinates, ",")
	result := [2]float64{0, 0}

	if len(coords) != 2 {
		return result, errors.New("invalid coordinates given. format must be ['lat', 'lng']")
	}
	lat := ToFloat(coords[0])
	lng := ToFloat(coords[1])
	result[0] = lat
	result[1] = lng
	return result, nil
}

func CoordsToString(coords [2]float64) string {
	return FromFloat(coords[0]) + "," + FromFloat(coords[1])
}
