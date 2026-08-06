package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/joho/godotenv"

	"seanboaden.dev/fuel/internal/cron"
	"seanboaden.dev/fuel/internal/secrets"
)

var cronSecrets = []string{"NSW_TAS_API_KEY", "NSW_TAS_API_SECRET", "SA_API_KEY", "QLD_API_KEY"}

func handler(ctx context.Context, evt cron.Event) error {
	return cron.Run(ctx, evt)
}

func main() {
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		if err := secrets.Load(cronSecrets); err != nil {
			log.Fatalf("secrets: %v", err)
		}
		lambda.Start(handler)
		return
	}

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	if err := secrets.Load(cronSecrets); err != nil {
		log.Fatalf("secrets: %v", err)
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
