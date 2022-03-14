package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"example.com/fuel/types"
	"example.com/fuel/util"
	"github.com/gorilla/mux"
)

func GetNearest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// Get fuel
	var items, date = util.GetFuelPrices()

	// Set the header - json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Parse coordinates string -> [2]float64
	coordinates := util.ParseCoordinates(vars["coordinates"])
	lat1 := util.ToFloat(coordinates[0])
	lng1 := util.ToFloat(coordinates[1])
	userCoordinates := [2]float64{lat1, lng1}

	// Iterate items - add the haversine distance between user coordinates and the fuel station
	for idx := range items {
		stationCoordinates := [2]float64{util.ToFloat(items[idx].Latitude), util.ToFloat(items[idx].Longitude)}

		items[idx].DistanceTo = util.GetDistance(userCoordinates, stationCoordinates)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].DistanceTo < items[j].DistanceTo
	})

	for i := 0; i < 5; i++ {
		var origin = util.CoordsToString(userCoordinates)
		var destination = util.CoordsToString([2]float64{util.ToFloat(items[i].Latitude), util.ToFloat(items[i].Longitude)})
		distance, duration := util.GetJourney(origin, destination)
		items[i].JourneyDistance = distance
		items[i].JourneyTime = duration
	}

	// Group date and items into struct for json encode
	type Json struct {
		Date     string
		Stations []types.Item
	}
	_json := Json{Stations: items, Date: strings.Fields(date)[0]}

	// Write to response
	json.NewEncoder(w).Encode(_json)
}

/*
	Returns the cheapest within a certain radius
*/
func GetCheapest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// Get fuel
	var items, date = util.GetFuelPrices()

	// Set the header - json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Parse coordinates string -> [2]float64
	coordinates := util.ParseCoordinates(vars["coordinates"])
	lat1 := util.ToFloat(coordinates[0])
	lng1 := util.ToFloat(coordinates[1])
	userCoordinates := [2]float64{lat1, lng1}

	itemsWithinRadius := []types.Item{}

	// If the distanceTo isn't within radius, skip
	for idx := range items {
		stationCoordinates := [2]float64{util.ToFloat(items[idx].Latitude), util.ToFloat(items[idx].Longitude)}

		distanceTo := util.GetDistance(userCoordinates, stationCoordinates)

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
		var origin = util.CoordsToString(userCoordinates)
		var destination = util.CoordsToString([2]float64{util.ToFloat(itemsWithinRadius[i].Latitude), util.ToFloat(itemsWithinRadius[i].Longitude)})
		distance, duration := util.GetJourney(origin, destination)
		itemsWithinRadius[i].JourneyDistance = distance
		itemsWithinRadius[i].JourneyTime = duration
	}

	// Group date and items into struct for json encode
	type Json struct {
		Date     string
		Stations []types.Item
	}
	_json := Json{Stations: itemsWithinRadius, Date: strings.Fields(date)[0]}

	// Write to response
	json.NewEncoder(w).Encode(_json)
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/nearest/{coordinates}", GetNearest).Methods("GET")
	r.HandleFunc("/cheapest/{coordinates}", GetCheapest).Methods("GET").Queries("radius", "{radius:[0-9]+}")

	// Error - need radius query
	r.HandleFunc("/cheapest/{coordinates}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Missing ?radius={value} in URL\n"))
	}).Methods("GET")

	fmt.Println("Serving requests on localhost:" + os.Getenv("PORT"))

	http.ListenAndServe(":"+os.Getenv("PORT"), r)
}
