package util

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

/*
 * Parse coordinates URL param
 */
func ParseCoordinates(coordinates string) ([2]float64, error) {
	coords := strings.Split(coordinates, ",")
	result := [2]float64{0, 0}

	if len(coords) != 2 {
		return result, errors.New("invalid coordinates given. format must be ['lat', 'lng']")
	}
	lat := ToFloat(coords[0])
	lng := ToFloat(coords[1])
	result[0] = lat
	result[1] = lng
	return result, nil
}

func CoordsToString(coords [2]float64) string {
	return FromFloat(coords[0]) + "," + FromFloat(coords[1])
}

func ExtractCoordinates(c *gin.Context) ([2]float64, [2]float64, error) {
	tlQ := c.Query("topLeft")
	brQ := c.Query("bottomRight")

	tl, err := ParseCoordinates(tlQ)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid topLeft: " + err.Error()})
		return [2]float64{}, [2]float64{}, err
	}

	br, err := ParseCoordinates(brQ)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bottomRight: " + err.Error()})
		return [2]float64{}, [2]float64{}, err
	}

	if tl[0] <= br[0] || tl[1] >= br[1] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bounding box: topLeft must be north-west of bottomRight"})
		return [2]float64{}, [2]float64{}, err
	}

	return tl, br, nil
}