package routes

import (
	"errors"
	"sort"
	"strings"

	"example.com/fuel/types"
	"example.com/fuel/util"
	"github.com/gin-gonic/gin"
)

/*
 * Returns the nearest to given coordinates
 */
func GetNearest(c *gin.Context) {
	coords, success := c.Params.Get("coordinates")
	if !success {
		panic(errors.New("unable to retrieve coordinates"))
	}

	// Parse coordinates string -> [2]float64
	coordinates, err := util.ParseCoordinates(coords)
	if err != nil {
		panic(err)
	}

	// Get fuel
	var items, date = util.GetFuelPrices()

	// Iterate items - add the haversine distance between user coordinates and the fuel station
	for idx := range items {
		stationCoordinates := [2]float64{util.ToFloat(items[idx].Latitude), util.ToFloat(items[idx].Longitude)}

		items[idx].DistanceTo = util.GetDistance(coordinates, stationCoordinates)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].DistanceTo < items[j].DistanceTo
	})

	for i := 0; i < 5; i++ {
		var origin = util.CoordsToString(coordinates)
		var destination = util.CoordsToString([2]float64{util.ToFloat(items[i].Latitude), util.ToFloat(items[i].Longitude)})
		distance, duration := util.GetJourney(origin, destination)
		items[i].JourneyDistance = distance
		items[i].JourneyTime = duration
	}

	// Group date and items into struct for json encode
	response := types.JsonResponse{Stations: items, Date: strings.Fields(date)[0]}

	c.JSON(200, response)
}
