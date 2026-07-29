Go

- Gin
- Air for live reload

Run Dev Server:

```
air
```

## Architecture

### Folder structure

- `cmd/api/dev` — local Gin server entry point
- `cmd/api/prod` — Lambda + gin-api-proxy entry point (API Gateway)
- `cmd/cron` — refresh job: fetches from providers, writes to DynamoDB
- `internal/providers` — upstream fuel price fetchers
  - `wa.go` — WA FuelWatch (RSS/XML)
  - `nsw_tas.go` — NSW+TAS FuelAPI (OAuth bearer + apikey headers)
  - `sa_qld.go` — SA + QLD Fuel Pricing Information Scheme
  - `providers.go` — `FetchAllStations()` parallel aggregator
- `internal/routes` — Gin route handlers (`current.go`, `journey.go`, `health.go`)
- `internal/store` — DynamoDB persistence (`stations.go` → `PutStations`)
- `internal/types` — shared domain types (`Station`, `StationItem`, `FuelPrice`)
- `internal/util`
  - `geohash/` — `CreateGeohash(Station) StationItem` (DynamoDB key generation)
  - `haversine.go`, `conversions.go`, `parse.go` — misc helpers
- `internal/auth` — `nsw_tas.go` (OAuth token fetch + cache in `auth.json`)

### DynamoDB table: `Stations`

Env var `DDB_TABLE_STATIONS` selects the table name.

| Key         | Field             | Composition             | Purpose                                 |
|-------------|-------------------|-------------------------|-----------------------------------------|
| PK          | `RegionGeohash`   | `<shard 1-10>#<p1>`    | World / region queries (sharded)        |
| SK          | `TownGeohash`     | `<p8>#<station_id>`    | Fine-grained range + uniqueness        |
| GSI1 PK     | `SubRegionGeohash`| `<p4>`                 | City-level queries (no shard)          |
| GSI1 SK     | `TownGeohash`     | `<p8>#<station_id>`    | Town-level `begins_with` queries       |

Other attributes: `StationId`, `Title`, `Brand`, `Address`, `Latitude`,
`Longitude`, `Ulp91`, `Ulp95`, `Ulp98`, `Diesel`, `Date`.

Query patterns:

- **PK only** → fan out 10 parallel queries (one per shard), truncate.
  Used for Australia-wide map (truncated subset).
- **PK + SK** → region/bbox queries, SK range-filtered by geohash prefix.
- **GSI1PK only** → city-level, cover bbox with P4 cells.
- **GSI1PK + GSI1SK (begins_with)** → town-level, P4 + P6/P7 prefix.

### Write path

`cmd/cron/main.go` is the refresh job: calls `providers.FetchAllStations()` →
`store.PutStations(items)`. `PutStations` dedupes by `StationId` (first-wins),
marshals each via `geohash.CreateGeohash` + `attributevalue.MarshalMap`, chunks
into 25-item `BatchWriteItem` batches (DynamoDB hard limit), writes sequentially
with a retry loop for throttled / unprocessed items (exponential backoff,
100ms→5s cap, 10 attempts).

Throttle note: the table is provisioned-capacity; per-cron runtime is bound by
`items / WriteCapacityUnits-per-sec`. For large providers under low WCU, raise
`WriteCapacityUnits` or switch the table to on-demand.

### Environment variables (see `.env.example`)

- `BASE_PATH` — API route prefix (Lambda)
- `ENVIRONMENT` — `local` / `prod`
- `MAPS_KEY` — Google Maps API key (journey endpoint)
- `NSW_TAS_API_KEY`, `NSW_TAS_API_SECRET` — OneGov NSW FuelAPI creds
- `SA_API_KEY`, `QLD_API_KEY` — SA / QLD Fuel Pricing Information Scheme keys
- `DDB_TABLE_STATIONS` — DynamoDB Stations table name

