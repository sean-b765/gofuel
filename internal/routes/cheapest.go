package routes

import (
	"errors"
	"slices"

	"github.com/gin-gonic/gin"
	"seanboaden.dev/fuel/internal/providers"
	"seanboaden.dev/fuel/internal/types"
	"seanboaden.dev/fuel/internal/util"
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
	items := providers.GetWaPricesCurrent()

	nswTas := providers.GetNswTasPricesCurrent()
	println(len(nswTas))
	items = slices.Concat(items, nswTas)

	itemsWithinRadius := []types.Station{}

	// If the distanceTo isn't within radius, skip
	for idx := range items {
		stationCoordinates := [2]float64{items[idx].Latitude, items[idx].Longitude}

		distanceTo := util.GetDistance(coordinates, stationCoordinates)

		if distanceTo > util.ToFloat(radius) {
			continue
		}

		items[idx].DistanceTo = distanceTo

		itemsWithinRadius = append(itemsWithinRadius, items[idx])
	}

	// sort.Slice(itemsWithinRadius, func(i, j int) bool {
	// 	return itemsWithinRadius[i].Price < itemsWithinRadius[j].Price
	// })

	// Group date and items into struct for json encode
	response := types.JsonResponse{Stations: itemsWithinRadius}

	c.Header("Access-Control-Allow-Origin", "*")
	c.JSON(200, response)
}
