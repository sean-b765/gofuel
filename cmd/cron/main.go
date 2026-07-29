package main

import (
	"log"

	"github.com/joho/godotenv"

	"seanboaden.dev/fuel/internal/providers"
	"seanboaden.dev/fuel/internal/store"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	items := providers.FetchAllStations()
	log.Printf("fetched %d stations", len(items))

	if err := store.PutStations(items); err != nil {
		log.Fatalf("error writing stations: %v", err)
	}

	log.Println("done")
}
