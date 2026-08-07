package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
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
	batchSize     = 25
	maxAttempts   = 10
	maxQueries    = 512
	maxConcurrent = 20
	gsiSubRegion  = "GSI_SubRegion"
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

	precision := geohash.PrecisionForBounds(tl, br)
	coverStart := time.Now()
	cells := geohash.CoveringHashes(tl[0], tl[1], br[0], br[1], uint(precision))
	log.Printf("[current] covering: p%d -> %d cells in %v", precision, len(cells), time.Since(coverStart))

	specs, err := buildQueries(precision, cells)
	if err != nil {
		return nil, err
	}
	log.Printf("[current] queries: %d @ p%d", len(specs), precision)

	items, err := runQueries(tableName, specs)
	if err != nil {
		return nil, err
	}

	filterStart := time.Now()
	stations := make([]types.Station, 0, len(items))
	for _, it := range items {
		if it.Latitude > tl[0] || it.Latitude < br[0] || it.Longitude < tl[1] || it.Longitude > br[1] {
			continue
		}
		stations = append(stations, toStation(it))
	}
	log.Printf("[current] filter: %d in / %d out in %v", len(items), len(stations), time.Since(filterStart))

	return stations, nil
}

type qspec struct {
	label string
	pk    string
	sk    string // begins_with prefix; "" => none
	gsi   bool
}

func shardsForPrecision(precision int) int {
	switch precision {
	case 1:
		return 2
	case 2:
		return 4
	case 3:
		return 8
	default: // p4
		return geohash.ShardCount
	}
}

func buildQueries(precision int, cells []string) ([]qspec, error) {
	var specs []qspec
	shards := shardsForPrecision(precision)

	switch precision {
	case 4:
		for _, c := range cells {
			specs = append(specs, qspec{label: "gsi:" + c, pk: c, gsi: true})
		}
	case 1:
		for _, c := range cells {
			for shard := 1; shard <= shards; shard++ {
				specs = append(specs, qspec{label: fmt.Sprintf("p1:%s/%d", c, shard), pk: geohash.RegionKey(shard, c)})
			}
		}
	default: // p2, p3
		for _, c := range cells {
			parent := c[:1]
			for shard := 1; shard <= shards; shard++ {
				specs = append(specs, qspec{label: fmt.Sprintf("p%d:%s/%d", precision, c, shard), pk: geohash.RegionKey(shard, parent), sk: c})
			}
		}
	}

	if len(specs) > maxQueries {
		return nil, ErrBoundsTooLarge
	}
	return specs, nil
}

func runQueries(tableName string, specs []qspec) ([]types.StationItem, error) {
	start := time.Now()
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	items := []types.StationItem{}

	for i, s := range specs {
		wg.Add(1)
		go func(i int, s qspec) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			qs := time.Now()
			var got []types.StationItem
			var err error
			if s.gsi {
				got, err = queryGSI(tableName, s.pk)
			} else {
				got, err = queryBase(tableName, s.pk, s.sk)
			}
			if err != nil {
				log.Printf("[current] query %d/%d %s: error after %v: %v", i+1, len(specs), s.label, time.Since(qs), err)
			} else {
				log.Printf("[current] query %d/%d %s: %d items in %v", i+1, len(specs), s.label, len(got), time.Since(qs))
			}

			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			items = append(items, got...)
		}(i, s)
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	log.Printf("[current] fan-out: %d items from %d queries in %v", len(items), len(specs), time.Since(start))
	return items, nil
}

func queryGSI(tableName, hash string) ([]types.StationItem, error) {
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
			return nil, fmt.Errorf("query gsi %s: %w", hash, err)
		}

		page := []types.StationItem{}
		if err := attributevalue.UnmarshalListOfMaps(out.Items, &page); err != nil {
			return nil, fmt.Errorf("unmarshal gsi %s: %w", hash, err)
		}
		items = append(items, page...)

		if len(out.LastEvaluatedKey) == 0 {
			return items, nil
		}
		startKey = out.LastEvaluatedKey
	}
}

func queryBase(tableName, pk, skPrefix string) ([]types.StationItem, error) {
	keyCond := "RegionGeohash = :pk"
	vals := map[string]awstypes.AttributeValue{
		":pk": &awstypes.AttributeValueMemberS{Value: pk},
	}
	if skPrefix != "" {
		keyCond += " AND begins_with(TownGeohash, :sk)"
		vals[":sk"] = &awstypes.AttributeValueMemberS{Value: skPrefix}
	}

	out, err := client.Query(context.Background(), &dynamodb.QueryInput{
		TableName:                 aws.String(tableName),
		KeyConditionExpression:    aws.String(keyCond),
		ExpressionAttributeValues: vals,
	})
	if err != nil {
		return nil, fmt.Errorf("query base %s: %w", pk, err)
	}

	page := []types.StationItem{}
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &page); err != nil {
		return nil, fmt.Errorf("unmarshal base %s: %w", pk, err)
	}
	return page, nil
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

func PutStations(stations []types.Station) ([]types.StationItem, error) {
	tableName := os.Getenv("DDB_TABLE_STATIONS")
	if tableName == "" {
		return nil, fmt.Errorf("DDB_TABLE_STATIONS not set")
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
			return nil, err
		}
	}

	return items, nil
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
