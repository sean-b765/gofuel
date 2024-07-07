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
 * Returns the cheapest within a certain radius
 */
func GetCheapest(c *gin.Context) {
	coords, success := c.Params.Get("coordinates")
	radius, _ := c.GetQuery("radius")

	if !success {
		panic(errors.New("unable to retrieve coordinates"))
	}

	// Parse coordinates string -> [2]float64
	coordinates, err := util.ParseCoordinates(coords)
	if err != nil {
		panic(err)
	}

	// Get fuel data
	var items, date = util.GetFuelPrices()

	itemsWithinRadius := []types.Item{}

	// If the distanceTo isn't within radius, skip
	for idx := range items {
		stationCoordinates := [2]float64{util.ToFloat(items[idx].Latitude), util.ToFloat(items[idx].Longitude)}

		distanceTo := util.GetDistance(coordinates, stationCoordinates)

		if distanceTo > util.ToFloat(radius) {
			continue
		}

		items[idx].DistanceTo = distanceTo

		itemsWithinRadius = append(itemsWithinRadius, items[idx])
	}

	sort.Slice(itemsWithinRadius, func(i, j int) bool {
		return itemsWithinRadius[i].Price < itemsWithinRadius[j].Price
	})

	for i := 0; i < 5; i++ {
		var origin = util.CoordsToString(coordinates)
		var destination = util.CoordsToString([2]float64{util.ToFloat(itemsWithinRadius[i].Latitude), util.ToFloat(itemsWithinRadius[i].Longitude)})
		distance, duration := util.GetJourney(origin, destination)
		itemsWithinRadius[i].JourneyDistance = distance
		itemsWithinRadius[i].JourneyTime = duration
	}

	// Group date and items into struct for json encode
	response := types.JsonResponse{Stations: itemsWithinRadius, Date: strings.Fields(date)[0]}

	c.Header("Access-Control-Allow-Origin", "*")
	c.JSON(200, response)
}
