package athena

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	athtypes "github.com/aws/aws-sdk-go-v2/service/athena/types"

	"seanboaden.dev/fuel/internal/types"
)

const (
	database  = "gofuel"
	table     = "stations"
	workGroup = "gofuel"

	dataStart = "2017/01/01"

	pollInterval = 250 * time.Millisecond
	queryTimeout = 25 * time.Second
)

var (
	clientOnce sync.Once
	client     *athena.Client
)

func getClient() *athena.Client {
	clientOnce.Do(func() {
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			log.Printf("[athena] unable to load AWS config: %v", err)
			return
		}
		client = athena.NewFromConfig(cfg)
	})
	return client
}

/*
 * Returns all stations recorded for a day, given as yyyy-mm-dd
 */
func GetStationsForDate(date string) ([]types.Station, error) {
	c := getClient()
	if c == nil {
		return nil, fmt.Errorf("athena client unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	query := fmt.Sprintf(
		`SELECT station_id, title, brand, address, "date", latitude, longitude, ulp91, ulp95, ulp98, diesel FROM "%s"."%s" WHERE dt = '%s'`,
		database, table, strings.ReplaceAll(date, "-", "/"),
	)

	out, err := c.StartQueryExecution(ctx, &athena.StartQueryExecutionInput{
		QueryString: aws.String(query),
		QueryExecutionContext: &athtypes.QueryExecutionContext{
			Database: aws.String(database),
		},
		WorkGroup: aws.String(workGroup),
	})
	if err != nil {
		return nil, fmt.Errorf("start query: %w", err)
	}
	id := out.QueryExecutionId

	if err := waitForQuery(ctx, c, id); err != nil {
		return nil, err
	}
	return fetchResults(ctx, c, id)
}

/*
 * Returns a station's price history between two dates, given as yyyy-mm-dd
 */
func GetStationHistory(stationID, from, to string) ([]types.Station, error) {
	if strings.ContainsAny(stationID, "'\";\\") {
		return nil, fmt.Errorf("invalid station id %q", stationID)
	}

	c := getClient()
	if c == nil {
		return nil, fmt.Errorf("athena client unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	fromDay, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, fmt.Errorf("parse from %q: %w", from, err)
	}
	// widen the partition range by a day so partitions holding "tomorrow" prices (WA) aren't missed
	dtFrom := fromDay.AddDate(0, 0, -1).Format("2006/01/02")
	if dtFrom < dataStart {
		dtFrom = dataStart
	}
	dtTo := strings.ReplaceAll(to, "-", "/")

	query := fmt.Sprintf(
		`SELECT station_id, title, brand, address, "date", latitude, longitude, ulp91, ulp95, ulp98, diesel FROM "%s"."%s" WHERE station_id = '%s' AND dt BETWEEN '%s' AND '%s' AND "date" BETWEEN '%s' AND '%s' ORDER BY "date" ASC`,
		database, table, stationID, dtFrom, dtTo, from, to,
	)

	out, err := c.StartQueryExecution(ctx, &athena.StartQueryExecutionInput{
		QueryString: aws.String(query),
		QueryExecutionContext: &athtypes.QueryExecutionContext{
			Database: aws.String(database),
		},
		WorkGroup: aws.String(workGroup),
	})
	if err != nil {
		return nil, fmt.Errorf("start query: %w", err)
	}
	id := out.QueryExecutionId

	if err := waitForQuery(ctx, c, id); err != nil {
		return nil, err
	}
	return fetchResults(ctx, c, id)
}

func waitForQuery(ctx context.Context, c *athena.Client, id *string) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		out, err := c.GetQueryExecution(ctx, &athena.GetQueryExecutionInput{QueryExecutionId: id})
		if err != nil {
			return fmt.Errorf("get query execution: %w", err)
		}
		status := out.QueryExecution.Status
		switch status.State {
		case athtypes.QueryExecutionStateSucceeded:
			if st := out.QueryExecution.Statistics; st != nil && st.DataScannedInBytes != nil {
				log.Printf("[athena] query %s scanned %d bytes", aws.ToString(id), *st.DataScannedInBytes)
			}
			return nil
		case athtypes.QueryExecutionStateFailed, athtypes.QueryExecutionStateCancelled:
			return fmt.Errorf("query %s: %s", strings.ToLower(string(status.State)), aws.ToString(status.StateChangeReason))
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			stopQuery(c, id)
			return fmt.Errorf("query timed out after %v", queryTimeout)
		}
	}
}

func stopQuery(c *athena.Client, id *string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := c.StopQueryExecution(ctx, &athena.StopQueryExecutionInput{QueryExecutionId: id}); err != nil {
		log.Printf("[athena] stop query %s: %v", aws.ToString(id), err)
	}
}

func fetchResults(ctx context.Context, c *athena.Client, id *string) ([]types.Station, error) {
	p := athena.NewGetQueryResultsPaginator(c, &athena.GetQueryResultsInput{
		QueryExecutionId: id,
	}, func(o *athena.GetQueryResultsPaginatorOptions) {
		o.Limit = 1000
	})

	stations := []types.Station{}
	var columns []string
	headerSkipped := false
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("get query results: %w", err)
		}
		rs := page.ResultSet
		if rs == nil || len(rs.Rows) == 0 {
			continue
		}

		if columns == nil && rs.ResultSetMetadata != nil {
			for _, col := range rs.ResultSetMetadata.ColumnInfo {
				columns = append(columns, aws.ToString(col.Name))
			}
		}

		rows := rs.Rows
		if !headerSkipped {
			rows = rows[1:]
			headerSkipped = true
		}

		for _, row := range rows {
			if len(row.Data) != len(columns) {
				return nil, fmt.Errorf("row has %d values, expected %d", len(row.Data), len(columns))
			}
			vals := make(map[string]string, len(columns))
			for i, col := range columns {
				vals[col] = aws.ToString(row.Data[i].VarCharValue)
			}
			station, err := toStation(vals)
			if err != nil {
				return nil, err
			}
			stations = append(stations, station)
		}
	}
	return stations, nil
}

func toStation(vals map[string]string) (types.Station, error) {
	latitude, err := parseFloat(vals, "latitude", 64)
	if err != nil {
		return types.Station{}, err
	}
	longitude, err := parseFloat(vals, "longitude", 64)
	if err != nil {
		return types.Station{}, err
	}
	ulp91, err := parseFloat(vals, "ulp91", 32)
	if err != nil {
		return types.Station{}, err
	}
	ulp95, err := parseFloat(vals, "ulp95", 32)
	if err != nil {
		return types.Station{}, err
	}
	ulp98, err := parseFloat(vals, "ulp98", 32)
	if err != nil {
		return types.Station{}, err
	}
	diesel, err := parseFloat(vals, "diesel", 32)
	if err != nil {
		return types.Station{}, err
	}

	return types.Station{
		Id:        vals["station_id"],
		Title:     vals["title"],
		Brand:     vals["brand"],
		Address:   vals["address"],
		Date:      vals["date"],
		Latitude:  latitude,
		Longitude: longitude,
		Price: types.FuelPrice{
			Ulp91:  float32(ulp91),
			Ulp95:  float32(ulp95),
			Ulp98:  float32(ulp98),
			Diesel: float32(diesel),
		},
	}, nil
}

func parseFloat(vals map[string]string, key string, bitSize int) (float64, error) {
	s := vals[key]
	if s == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(s, bitSize)
	if err != nil {
		return 0, fmt.Errorf("parse %s %q for station %s: %w", key, s, vals["station_id"], err)
	}
	return f, nil
}
