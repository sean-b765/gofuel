package routes

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"seanboaden.dev/fuel/internal/athena"
	"seanboaden.dev/fuel/internal/types"
)

/*
 * Returns all stations for a given day (yyyy-mm-dd)
 */
func GetDay(c *gin.Context) {
	start := time.Now()

	day, err := time.Parse("2006-01-02", c.Param("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date, expected yyyy-mm-dd"})
		return
	}
	if day.After(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date cannot be in the future"})
		return
	}

	stations, err := athena.GetStationsForDate(day.Format("2006-01-02"))
	if err != nil {
		log.Printf("[day] error after %v: %v", time.Since(start), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stations"})
		return
	}

	log.Printf("[day] done: %d stations in %v", len(stations), time.Since(start))
	c.JSON(http.StatusOK, types.JsonResponse{Stations: stations})
}
