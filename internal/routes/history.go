package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"seanboaden.dev/fuel/internal/athena"
	"seanboaden.dev/fuel/internal/s3"
	"seanboaden.dev/fuel/internal/types"
)

const (
	dateLayout    = "2006-01-02"
	maxHistoryDay = 365
	defaultDays   = 7
)

/*
 * Returns a station's price history between ?from= and ?to= (yyyy-mm-dd)
 */
func GetHistory(c *gin.Context) {
	start := time.Now()

	id := c.Param("id")
	if id == "" || len(id) > 64 || strings.ContainsAny(id, "'\";\\") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid station id"})
		return
	}

	now := time.Now()
	from, err := historyDate("from", c.Query("from"), now.AddDate(0, 0, -defaultDays))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	to, err := historyDate("to", c.Query("to"), now)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if from.After(to) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from must be on or before to"})
		return
	}
	if to.After(now) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to cannot be in the future"})
		return
	}
	if to.Sub(from).Hours()/24 > maxHistoryDay {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("range cannot exceed %d days", maxHistoryDay)})
		return
	}

	fromStr, toStr := from.Format(dateLayout), to.Format(dateLayout)

	// fully-past ranges are immutable: serve from cache when possible
	key := fmt.Sprintf("cache/history/%s/%s_%s.json", id, fromStr, toStr)
	cacheable := toStr != now.Format(dateLayout)
	if cacheable {
		exists, err := s3.Exists(key)
		if exists && err == nil {
			bytes, err := s3.Get(key)
			if err == nil {
				c.Data(http.StatusOK, "application/json; charset=utf-8", bytes)
				return
			}
		}
	}

	stations, err := athena.GetStationHistory(id, fromStr, toStr)
	if err != nil {
		log.Printf("[history] error after %v: %v", time.Since(start), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load station history"})
		return
	}

	if cacheable {
		if bytes, err := json.Marshal(stations); err == nil {
			if err := s3.Put(key, bytes); err != nil {
				log.Printf("[history] cache write %s: %v", key, err)
			}
		}
	}

	log.Printf("[history] done: %d rows in %v", len(stations), time.Since(start))
	c.JSON(http.StatusOK, types.JsonResponse{Stations: stations})
}

func historyDate(label, value string, fallback time.Time) (time.Time, error) {
	if value == "" {
		return fallback, nil
	}
	t, err := time.Parse(dateLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s, expected yyyy-mm-dd", label)
	}
	return t, nil
}
