package util

import (
	"fmt"
	"net/http"
	"os"
)

func GetJourney(origin, destination string) {
	url := "https://maps.googleapis.com/maps/api/distancematrix/json?origins=" + origin + "&destinations=" + destination + "&mode=driving&language=en-US&key=" + os.Getenv("MAPS_KEY")
	resp, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Body)
	fmt.Println(url)
}
