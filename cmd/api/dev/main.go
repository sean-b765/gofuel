package main

import (
	"seanboaden.dev/fuel/internal/routes"
)

func main() {
	r := routes.SetupRouter()
	// Local entry point
	r.Run(":8081")
}
