package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"seanboaden.dev/fuel/internal/athena"
	"seanboaden.dev/fuel/internal/s3"
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

	date := day.Format("2006-01-02")
	
	// Look in cache
	key := fmt.Sprintf("cache/day/%v.json", date)
	exists, err := s3.Exists(key)
	if exists && err == nil {
		bytes, err := s3.Get(key)
		if err == nil {
			c.Data(http.StatusOK, "application/json; charset=utf-8", bytes)
			return
		}
	}

	stations, err := athena.GetStationsForDate(date)
	if err != nil {
		log.Printf("[day] error after %v: %v", time.Since(start), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stations"})
		return
	}

	// Write through to s3 as cache
	bytes, err := json.Marshal(stations)
	if err == nil {
	fmt.Printf("[s3] wrote %v to s3 %v %v\n", len(stations), key, err)
		s3.Put(key, bytes)
	}

	log.Printf("[day] done: %d stations in %v", len(stations), time.Since(start))
	c.JSON(http.StatusOK, types.JsonResponse{Stations: stations})
}
