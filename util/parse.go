package util

import "strings"

func ParseCoordinates(coordinates string) []string {
	return strings.Split(coordinates, ",")
}

func CoordsToString(coords [2]float64) string {
	return FromFloat(coords[0]) + "," + FromFloat(coords[1])
}
