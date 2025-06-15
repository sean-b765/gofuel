package main

import (
	"context"
	"os"
	"time"

	"example.com/fuel/routes"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

var adapter *ginadapter.GinLambda

func SetupRouter() (*gin.Engine) {
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // or your specific domain
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.GET("/nearest/:coordinates", routes.GetNearest)
	r.GET("/cheapest/:coordinates", routes.GetCheapest)
	r.GET("/journey", routes.GetJourney)

	return r
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return adapter.ProxyWithContext(ctx, req)
}

func IsProduction() (bool) {
	environment := os.Getenv("ENVIRONMENT")
	return environment == "production"
}

func main() {
	r := SetupRouter()

	// Create Adapter from router
	if IsProduction() {
		// Lambda entry point
		adapter = ginadapter.New(r)
		adapter.StripBasePath(os.Getenv("BASE_PATH"))
		lambda.Start(Handler)
	} else {
		// Local entry point
		r.Run(":8081")
	}
}