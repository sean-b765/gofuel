package util

import "math"

func GetDistance(coordinates1 [2]float64, coordinates2 [2]float64) float64 {
	R := 6371

	lat1 := coordinates1[0]
	lng1 := coordinates1[1]
	lat2 := coordinates2[0]
	lng2 := coordinates2[1]

	var deltaLat = degToRad(lat1 - lat2)
	var deltaLng = degToRad(lng1 - lng2)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) + math.Cos(degToRad(lat1))*math.Cos(degToRad(lat2))*math.Sin(deltaLng/2)*math.Sin(deltaLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return float64(R) * c
}

func degToRad(d float64) float64 {
	return d * math.Pi / 180
}
