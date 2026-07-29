package routes

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"seanboaden.dev/fuel/internal/store"
	"seanboaden.dev/fuel/internal/types"
	"seanboaden.dev/fuel/internal/util"
)

/*
 * Returns stations with current prices within a bounding box
 */
func GetCurrent(c *gin.Context) {
	tl, err := util.ParseCoordinates(c.Query("topLeft"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid topLeft: " + err.Error()})
		return
	}

	br, err := util.ParseCoordinates(c.Query("bottomRight"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bottomRight: " + err.Error()})
		return
	}

	if tl[0] <= br[0] || tl[1] >= br[1] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bounding box: topLeft must be north-west of bottomRight"})
		return
	}

	stations, err := store.GetStationsInBounds(tl, br)
	if errors.Is(err, store.ErrBoundsTooLarge) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stations"})
		return
	}

	c.JSON(http.StatusOK, types.JsonResponse{Stations: stations})
}
