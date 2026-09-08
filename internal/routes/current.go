package routes

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"seanboaden.dev/fuel/internal/store"
	"seanboaden.dev/fuel/internal/types"
	"seanboaden.dev/fuel/internal/util"
)

/*
 * Returns stations with current prices within a bounding box
 */
func GetCurrent(c *gin.Context) {
	start := time.Now()
	tl, br, err := util.ExtractCoordinates(c)
	if err != nil {
		return
	}

	stations, err := store.GetStationsInBounds(tl, br)
	if errors.Is(err, store.ErrBoundsTooLarge) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		log.Printf("[current] error after %v: %v", time.Since(start), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stations"})
		return
	}

	log.Printf("[current] done: %d stations in %v", len(stations), time.Since(start))
	c.JSON(http.StatusOK, types.JsonResponse{Stations: stations})
}
