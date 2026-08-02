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
- `internal/store` — DynamoDB persistence (`stations.go` → `PutStations`, `auth.go` → `GetNswTasToken`)
- `internal/types` — shared domain types (`Station`, `StationItem`, `FuelPrice`)
- `internal/util`
  - `geohash/` — `CreateGeohash(Station) StationItem` (DynamoDB key generation)
  - `haversine.go`, `conversions.go`, `parse.go` — misc helpers

### DynamoDB table: `Stations`

Env var `DDB_TABLE_STATIONS` selects the table name.

| Key     | Field              | Composition         | Purpose                          |
| ------- | ------------------ | ------------------- | -------------------------------- |
| PK      | `RegionGeohash`    | `<shard 1-10>#<p1>` | World / region queries (sharded) |
| SK      | `TownGeohash`      | `<p8>#<station_id>` | Fine-grained range + uniqueness  |
| GSI1 PK | `SubRegionGeohash` | `<p4>`              | City-level queries (no shard)    |
| GSI1 SK | `TownGeohash`      | `<p8>#<station_id>` | Town-level `begins_with` queries |

Other attributes: `StationId`, `Title`, `Brand`, `Address`, `Latitude`,
`Longitude`, `Ulp91`, `Ulp95`, `Ulp98`, `Diesel`, `Date`.

Query patterns:

`GET /current` picks a geohash precision from the bounding-box diagonal, then
fans out parallel DynamoDB queries (bounded concurrency) and filters to the
exact box in app code.

| diagonal    | precision | shards | path                                       |
|-------------|-----------|--------|--------------------------------------------|
| ≥ 2000 km   | p1        | 2/10   | base table, PK only                        |
| 400–2000 km | p2        | 4/10   | base table, PK + `begins_with(SK, p2cell)` |
| 80–400 km   | p3        | 8/10   | base table, PK + `begins_with(SK, p3cell)` |
| < 80 km     | p4        | 10/10  | `GSI_SubRegion`, PK EQ                     |

- **p4** (city): one `Query` per covering cell against `GSI_SubRegion`
  (`SubRegionGeohash = :cell`), no limit — small bbox, full result set.
- **p1–p3** (base table): for each covering cell, fan out across a subset of
  shards (shards are keyed by station-ID hash, so a subset is a
  geographically-distributed deterministic sample / virtual limit). PK is
  `<shard>#<p1>`; SK is `begins_with(TownGeohash, <cell>)` at p2/p3.
- All queries run in parallel via a bounded worker pool; results are filtered
  to the exact bounding box in app code. On-demand capacity absorbs spikes.

### DynamoDB table: `Providers-Auth`

Env var `DDB_TABLE_AUTH` selects the table name. Used by
`internal/store/auth.go` to cache the NSW/TAS OneGov access token so the OAuth
client-credentials flow runs only when the cached token is missing or expired.

| Key | Field     | Value        | Purpose                          |
| --- | --------- | ------------ | -------------------------------- |
| PK  | `provider` | `nsw_tas`    | Single-row cache per provider    |

Other attributes: `access_token`, `expires_in`, `issued_at`, `ttl`.

`ttl` = `issued_at/1000 + expires_in` (epoch seconds); DynamoDB TTL is enabled
on the `ttl` attribute, so expired tokens are eventually reaped automatically.
The read path (`store.GetNswTasToken`) still checks expiry in-app to avoid
using a token in the TTL grace window — if found and not expired it is reused,
otherwise a fresh token is fetched and overwritten via `PutItem`.

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
- `DDB_TABLE_AUTH` — DynamoDB auth table name (OAuth token cache; PK `provider`, TTL on `ttl`)
