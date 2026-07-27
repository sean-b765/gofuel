package main

import (
	"log"

	"github.com/joho/godotenv"
	"seanboaden.dev/fuel/internal/routes"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	r := routes.SetupRouter()
	// Local entry point
	r.Run(":8081")
}
