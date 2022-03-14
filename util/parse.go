package util

import "strings"

func ParseCoordinates(coordinates string) []string {
	return strings.Split(coordinates, ",")
}
