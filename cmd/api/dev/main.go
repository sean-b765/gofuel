package main

import (
	"log"

	"github.com/joho/godotenv"
	"seanboaden.dev/fuel/internal/routes"
	"seanboaden.dev/fuel/internal/secrets"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	if err := secrets.Load([]string{"MAPS_KEY"}); err != nil {
		log.Fatalf("secrets: %v", err)
	}
	r := routes.SetupRouter()
	// Local entry point
	r.Run(":8081")
}
