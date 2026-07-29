package geohash

import (
	"fmt"
	"hash/crc32"

	geohash "github.com/mmcloughlin/geohash"

	"seanboaden.dev/fuel/internal/types"
)

const shardCount = 10
const subRegionPrecision = 4

/*
 * Returns the precision-4 geohashes covering the bounding box
 * defined by top-left and bottom-right corners.
 */
func CoveringHashes(tlLat, tlLng, brLat, brLng float64) []string {
	start := geohash.BoundingBox(geohash.EncodeWithPrecision(tlLat, tlLng, subRegionPrecision))
	latStep := start.MaxLat - start.MinLat
	lngStep := start.MaxLng - start.MinLng

	seen := map[string]struct{}{}
	hashes := []string{}

	for lat := start.MinLat + latStep/2; lat+latStep/2 >= brLat; lat -= latStep {
		for lng := start.MinLng + lngStep/2; lng-lngStep/2 <= brLng; lng += lngStep {
			h := geohash.EncodeWithPrecision(lat, lng, subRegionPrecision)
			if _, ok := seen[h]; ok {
				continue
			}
			seen[h] = struct{}{}
			hashes = append(hashes, h)
		}
	}

	return hashes
}

func CreateGeohash(s types.Station) types.StationItem {
	p1 := geohash.EncodeWithPrecision(s.Latitude, s.Longitude, 1)
	p4 := geohash.EncodeWithPrecision(s.Latitude, s.Longitude, 4)
	p8 := geohash.EncodeWithPrecision(s.Latitude, s.Longitude, 8)
	shard := crc32.ChecksumIEEE([]byte(s.Id))%uint32(shardCount) + 1

	return types.StationItem{
		RegionGeohash:    fmt.Sprintf("%d#%s", shard, p1),
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
