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
- `cmd/cron` — refresh job: dual-mode (Lambda via EventBridge Scheduler, or local via `PROVIDER`/`DAY` env)
- `internal/cron` — `cron.go` (`Event` + `Run`): validates payload, fetches one provider, writes to DynamoDB
- `internal/providers` — upstream fuel price fetchers
  - `wa.go` — WA FuelWatch (RSS/XML); `GetWaPrices(day)` where day is `""` (today) or `"tomorrow"`
  - `nsw_tas.go` — NSW+TAS FuelAPI (OAuth bearer + apikey headers)
  - `sa_qld.go` — SA + QLD Fuel Pricing Information Scheme
  - `providers.go` — `FetchAllStations()` parallel aggregator + `FetchProviderAndDay(provider, day)`
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

`cmd/cron/main.go` is the refresh entry point. In Lambda mode it receives an
`internal/cron.Event` from EventBridge Scheduler, validates it, calls
`providers.FetchProviderAndDay(provider, day)`, then `store.PutStations(items)`.
In local mode it reads `PROVIDER`/`DAY` env vars and does the same.

`PutStations` dedupes by `StationId` (first-wins), marshals each via
`geohash.CreateGeohash` + `attributevalue.MarshalMap`, chunks into 25-item
`BatchWriteItem` batches (DynamoDB hard limit), writes sequentially with a retry
loop for throttled / unprocessed items (exponential backoff, 100ms→5s cap,
10 attempts).

### Cron / EventBridge schedules

The cron Lambda (`gofuel-cron`) is invoked by EventBridge Scheduler. Each
schedule carries a constant JSON payload:

```json
{"provider": "wa", "day": ""}
```

`provider` is required (`wa` | `nsw_tas` | `sa_qld`). `day` is `""` (today) or
`"tomorrow"` (WA FuelWatch only; ignored by other providers).

All schedules use `schedule_expression_timezone = "Australia/Perth"`:

| Schedule | Expression | Payload | Cadence |
|----------|------------|---------|---------|
| `gofuel-cron-wa-tomorrow` | `cron(0 16 * * ? *)` | `{"provider":"wa","day":"tomorrow"}` | Daily, 4PM AWST |
| `gofuel-cron-nsw-tas` | `cron(0 5,9,12,15,17 * * ? *)` | `{"provider":"nsw_tas","day":""}` | 5×/day |
| `gofuel-cron-sa-qld` | `cron(0 5,9,12,15,17 * * ? *)` | `{"provider":"sa_qld","day":""}` | 5×/day |

Validation in `internal/cron.Run`: unknown `provider` or `day` returns an error
(the Lambda invocation is marked failed, so EventBridge retry policy applies).

Local run: `PROVIDER=wa DAY=tomorrow go run ./cmd/cron`.

### Docker (multi-target)

The `Dockerfile` builds either entry point via `--build-arg FUNCTION=...`:

- API (default): `docker build -t gofuel .`
- Cron: `docker build --build-arg FUNCTION=cron -t gofuel:cron-latest .`

Cron images are tagged `gofuel:cron-*` in the same ECR repo as the API.

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
- `PROVIDER` — (local cron only) provider to refresh (`wa` | `nsw_tas` | `sa_qld`)
- `DAY` — (local cron only) WA day param (`""` | `tomorrow`)
