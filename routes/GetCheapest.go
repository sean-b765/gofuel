package routes

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"example.com/fuel/types"
	"example.com/fuel/util"
	"github.com/gorilla/mux"
)

/*
 * Returns the cheapest within a certain radius
 */
func GetCheapest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// Get fuel
	var items, date = util.GetFuelPrices()

	// Set the header - json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Parse coordinates string -> [2]float64
	coordinates, err := util.ParseCoordinates(vars["coordinates"])
	if err != nil {
		panic(err)
	}

	itemsWithinRadius := []types.Item{}

	// If the distanceTo isn't within radius, skip
	for idx := range items {
		stationCoordinates := [2]float64{util.ToFloat(items[idx].Latitude), util.ToFloat(items[idx].Longitude)}

		distanceTo := util.GetDistance(coordinates, stationCoordinates)

		if distanceTo > util.ToFloat(vars["radius"]) {
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

	// Write to response
	json.NewEncoder(w).Encode(response)
}
