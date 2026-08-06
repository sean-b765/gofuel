package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/joho/godotenv"

	"seanboaden.dev/fuel/internal/cron"
)

func handler(ctx context.Context, evt cron.Event) error {
	return cron.Run(ctx, evt)
}

func main() {
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		lambda.Start(handler)
		return
	}

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	evt := cron.Event{
		Provider: os.Getenv("PROVIDER"),
		Day:      os.Getenv("DAY"),
	}
	if err := cron.Run(context.Background(), evt); err != nil {
		log.Fatalf("cron: %v", err)
	}
	log.Println("done")
}
