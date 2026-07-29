package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awstypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"seanboaden.dev/fuel/internal/types"
	"seanboaden.dev/fuel/internal/util/geohash"
)

const (
	batchSize    = 25
	maxAttempts  = 10
	maxCells     = 100
	gsiSubRegion = "GSI_SubRegion"
)

var ErrBoundsTooLarge = errors.New("bounding box too large")

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

func GetStationsInBounds(tl, br [2]float64) ([]types.Station, error) {
	tableName := os.Getenv("DDB_TABLE_STATIONS")
	if tableName == "" {
		return nil, fmt.Errorf("DDB_TABLE_STATIONS not set")
	}

	hashes := geohash.CoveringHashes(tl[0], tl[1], br[0], br[1])
	if len(hashes) > maxCells {
		return nil, ErrBoundsTooLarge
	}

	stations := []types.Station{}
	for _, h := range hashes {
		items, err := querySubRegion(tableName, h)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.Latitude > tl[0] || item.Latitude < br[0] || item.Longitude < tl[1] || item.Longitude > br[1] {
				continue
			}
			stations = append(stations, toStation(item))
		}
	}

	return stations, nil
}

func querySubRegion(tableName, hash string) ([]types.StationItem, error) {
	items := []types.StationItem{}
	var startKey map[string]awstypes.AttributeValue

	for {
		out, err := client.Query(context.Background(), &dynamodb.QueryInput{
			TableName:              aws.String(tableName),
			IndexName:              aws.String(gsiSubRegion),
			KeyConditionExpression: aws.String("SubRegionGeohash = :hash"),
			ExpressionAttributeValues: map[string]awstypes.AttributeValue{
				":hash": &awstypes.AttributeValueMemberS{Value: hash},
			},
			ExclusiveStartKey: startKey,
		})
		if err != nil {
			return nil, fmt.Errorf("query subregion %s: %w", hash, err)
		}

		page := []types.StationItem{}
		if err := attributevalue.UnmarshalListOfMaps(out.Items, &page); err != nil {
			return nil, fmt.Errorf("unmarshal subregion %s: %w", hash, err)
		}
		items = append(items, page...)

		if len(out.LastEvaluatedKey) == 0 {
			return items, nil
		}
		startKey = out.LastEvaluatedKey
	}
}

func toStation(item types.StationItem) types.Station {
	return types.Station{
		Id:        item.StationId,
		Title:     item.Title,
		Brand:     item.Brand,
		Address:   item.Address,
		Date:      item.Date,
		Latitude:  item.Latitude,
		Longitude: item.Longitude,
		Price: types.FuelPrice{
			Ulp91:  item.Ulp91,
			Ulp95:  item.Ulp95,
			Ulp98:  item.Ulp98,
			Diesel: item.Diesel,
		},
	}
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
