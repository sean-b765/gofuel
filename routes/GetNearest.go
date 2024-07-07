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
 * Returns the nearest to given coordinates
 */
func GetNearest(w http.ResponseWriter, r *http.Request) {
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

	// Iterate items - add the haversine distance between user coordinates and the fuel station
	for idx := range items {
		stationCoordinates := [2]float64{util.ToFloat(items[idx].Latitude), util.ToFloat(items[idx].Longitude)}

		items[idx].DistanceTo = util.GetDistance(coordinates, stationCoordinates)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].DistanceTo < items[j].DistanceTo
	})

	for i := 0; i < 5; i++ {
		var origin = util.CoordsToString(coordinates)
		var destination = util.CoordsToString([2]float64{util.ToFloat(items[i].Latitude), util.ToFloat(items[i].Longitude)})
		distance, duration := util.GetJourney(origin, destination)
		items[i].JourneyDistance = distance
		items[i].JourneyTime = duration
	}

	// Group date and items into struct for json encode
	response := types.JsonResponse{Stations: items, Date: strings.Fields(date)[0]}

	// Write to response
	json.NewEncoder(w).Encode(response)
}
