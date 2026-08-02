package geohash

import (
	"fmt"
	"hash/crc32"

	g "github.com/mmcloughlin/geohash"

	"seanboaden.dev/fuel/internal/types"
	"seanboaden.dev/fuel/internal/util"
)

const ShardCount = 10

const (
	diagP1 = 2000.0
	diagP2 = 400.0
	diagP3 = 80.0
)

/*
 * Returns the precision-4 geohashes covering the bounding box
 * defined by top-left and bottom-right corners.
 */
func CoveringHashes(tlLat, tlLng, brLat, brLng float64, precision uint) []string {
	start := g.BoundingBox(g.EncodeWithPrecision(tlLat, tlLng, precision))
	latStep := start.MaxLat - start.MinLat
	lngStep := start.MaxLng - start.MinLng

	seen := map[string]struct{}{}
	hashes := []string{}

	for lat := start.MinLat + latStep/2; lat+latStep/2 >= brLat; lat -= latStep {
		for lng := start.MinLng + lngStep/2; lng-lngStep/2 <= brLng; lng += lngStep {
			h := g.EncodeWithPrecision(lat, lng, precision)
			if _, ok := seen[h]; ok {
				continue
			}
			seen[h] = struct{}{}
			hashes = append(hashes, h)
		}
	}

	return hashes
}

/*
 * Picks a geohash precision for the bounding box diagonal,
 * per the read design:
 *   ≥ 2000 km → 1,  ≥ 400 km → 2,  ≥ 80 km → 3,  else 4.
 */
func PrecisionForBounds(tl, br [2]float64) int {
	d := util.GetDistance(tl, br)
	switch {
	case d >= diagP1:
		return 1
	case d >= diagP2:
		return 2
	case d >= diagP3:
		return 3
	default:
		return 4
	}
}

/*
 * RegionKey formats the base-table partition key for a shard + p1 prefix.
 */
func RegionKey(shard int, p1 string) string {
	return fmt.Sprintf("%d#%s", shard, p1)
}

func CreateGeohash(s types.Station) types.StationItem {
	p1 := g.EncodeWithPrecision(s.Latitude, s.Longitude, 1)
	p4 := g.EncodeWithPrecision(s.Latitude, s.Longitude, 4)
	p8 := g.EncodeWithPrecision(s.Latitude, s.Longitude, 8)
	shard := crc32.ChecksumIEEE([]byte(s.Id))%uint32(ShardCount) + 1

	return types.StationItem{
		RegionGeohash:    RegionKey(int(shard), p1),
		TownGeohash:      p8 + "#" + s.Id,
		SubRegionGeohash: p4,
		StationId:        s.Id,
		Title:            s.Title,
		Brand:            s.Brand,
		Address:          s.Address,
		Date:             s.Date,
		Latitude:         s.Latitude,
		Longitude:        s.Longitude,
		Ulp91:            s.Price.Ulp91,
		Ulp95:            s.Price.Ulp95,
		Ulp98:            s.Price.Ulp98,
		Diesel:           s.Price.Diesel,
	}
}