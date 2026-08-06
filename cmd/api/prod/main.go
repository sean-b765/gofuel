package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"seanboaden.dev/fuel/internal/routes"
	"seanboaden.dev/fuel/internal/secrets"
)

var adapter *ginadapter.GinLambda

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return adapter.ProxyWithContext(ctx, req)
}

func main() {
	if err := secrets.Load([]string{"MAPS_KEY"}); err != nil {
		log.Fatalf("secrets: %v", err)
	}
	gin.SetMode(gin.ReleaseMode)
	r := routes.SetupRouter()

	// Create Adapter from router
	// Lambda entry point
	adapter = ginadapter.New(r)
	adapter.StripBasePath(os.Getenv("BASE_PATH"))
	lambda.Start(Handler)
}
