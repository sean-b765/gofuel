package routes

import (
	"github.com/gin-gonic/gin"
	"seanboaden.dev/fuel/internal/types"
	"seanboaden.dev/fuel/internal/util"
	"seanboaden.dev/fuel/internal/util/google"
)

/*
 * Returns the cheapest within a certain radius
 */
func GetJourney(c *gin.Context) {
	originQuery := c.Query("origin")
	destinationQuery := c.Query("destination")

	// Parse coordinates string -> [2]float64
	originCoordinates, err := util.ParseCoordinates(originQuery)
	if err != nil {
		panic(err)
	}

	destinationCoordinates, err := util.ParseCoordinates(destinationQuery)
	if err != nil {
		panic(err)
	}

	origin := util.CoordsToString(originCoordinates)
	destination := util.CoordsToString(destinationCoordinates)
	distance, duration := google.GetJourney(origin, destination)

	response := types.GoogleJourneyResponse{Distance: distance, Duration: duration}

	c.Header("Access-Control-Allow-Origin", "*")
	c.JSON(200, response)
}
