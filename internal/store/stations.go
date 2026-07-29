package store

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awstypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"seanboaden.dev/fuel/internal/types"
	"seanboaden.dev/fuel/internal/util/geohash"
)

const (
	batchSize   = 25
	maxAttempts = 10
)

var (
	client     *dynamodb.Client
	maxBackoff = 5 * time.Second
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load AWS config: %v", err))
	}
	client = dynamodb.NewFromConfig(cfg)
}

func PutStations(stations []types.Station) error {
	tableName := os.Getenv("DDB_TABLE_STATIONS")
	if tableName == "" {
		return fmt.Errorf("DDB_TABLE_STATIONS not set")
	}

	seen := make(map[string]struct{}, len(stations))
	items := make([]types.StationItem, 0, len(stations))
	for _, s := range stations {
		if s.Id == "" {
			continue
		}
		if _, ok := seen[s.Id]; ok {
			continue
		}
		seen[s.Id] = struct{}{}
		items = append(items, geohash.CreateGeohash(s))
	}

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		if err := putBatch(tableName, items[i:end]); err != nil {
			return err
		}
	}

	return nil
}

func putBatch(tableName string, batch []types.StationItem) error {
	writeReqs := make([]awstypes.WriteRequest, len(batch))
	for i, item := range batch {
		av, err := attributevalue.MarshalMap(item)
		if err != nil {
			return fmt.Errorf("marshal station %s: %w", item.StationId, err)
		}
		writeReqs[i] = awstypes.WriteRequest{
			PutRequest: &awstypes.PutRequest{Item: av},
		}
	}

	requestItems := map[string][]awstypes.WriteRequest{tableName: writeReqs}
	backoff := 100 * time.Millisecond

	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := client.BatchWriteItem(context.Background(), &dynamodb.BatchWriteItemInput{
			RequestItems: requestItems,
		})
		if err != nil {
			if attempt == maxAttempts-1 {
				return fmt.Errorf("batch write (attempt %d): %w", attempt+1, err)
			}
			time.Sleep(backoff)
			backoff = nextBackoff(backoff)
			continue
		}

		if len(resp.UnprocessedItems) == 0 {
			return nil
		}

		requestItems = resp.UnprocessedItems
		time.Sleep(backoff)
		backoff = nextBackoff(backoff)
	}

	return nil
}

func nextBackoff(b time.Duration) time.Duration {
	b *= 2
	if b > maxBackoff {
		return maxBackoff
	}
	return b
}
