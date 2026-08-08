package firehose

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	fhtypes "github.com/aws/aws-sdk-go-v2/service/firehose/types"
	"seanboaden.dev/fuel/internal/types"
)

const (
	maxBatchRecords = 400
	maxAttempts     = 3
)

var (
	clientOnce sync.Once
	client     *firehose.Client
)

func getClient() *firehose.Client {
	clientOnce.Do(func() {
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			log.Printf("[firehose] unable to load AWS config: %v", err)
			return
		}
		client = firehose.NewFromConfig(cfg)
	})
	return client
}

func PutStations(items []types.StationItem) error {
	stream := os.Getenv("FIREHOSE_STREAM")
	if stream == "" {
		log.Printf("[firehose] FIREHOSE_STREAM not set; skipping (%d items)", len(items))
		return nil
	}

	c := getClient()
	if c == nil {
		return fmt.Errorf("firehose client unavailable")
	}

	if len(items) == 0 {
		return nil
	}

	total := 0
	for i := 0; i < len(items); i += maxBatchRecords {
		end := i + maxBatchRecords
		if end > len(items) {
			end = len(items)
		}
		n, err := putBatch(c, stream, items[i:end])
		if err != nil {
			return err
		}
		total += n
	}

	log.Printf("[firehose] sent=%d", total)
	return nil
}

func putBatch(c *firehose.Client, stream string, items []types.StationItem) (int, error) {
	records := make([]fhtypes.Record, 0, len(items))
	for _, it := range items {
		b, err := json.Marshal(it)
		if err != nil {
			return 0, fmt.Errorf("marshal station %s: %w", it.StationId, err)
		}
		b = append(b, '\n')
		records = append(records, fhtypes.Record{Data: b})
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		out, err := c.PutRecordBatch(context.Background(), &firehose.PutRecordBatchInput{
			DeliveryStreamName: aws.String(stream),
			Records:            records,
		})
		if err != nil {
			if attempt == maxAttempts-1 {
				return 0, fmt.Errorf("put record batch (attempt %d): %w", attempt+1, err)
			}
			continue
		}

		if aws.ToInt32(out.FailedPutCount) == 0 {
			return len(records), nil
		}

		var retry []fhtypes.Record
		for i, r := range out.RequestResponses {
			if r.RecordId == nil {
				retry = append(retry, records[i])
			}
		}
		records = retry
		if attempt == maxAttempts-1 {
			return 0, fmt.Errorf("firehose %d records failed after %d attempts", len(retry), maxAttempts)
		}
	}
	return len(records), nil
}

