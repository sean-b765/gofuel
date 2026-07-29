package geohash

import (
	"fmt"
	"hash/crc32"

	geohash "github.com/mmcloughlin/geohash"

	"seanboaden.dev/fuel/internal/types"
)

const shardCount = 10

func CreateGeohash(s types.Station) types.StationItem {
	p1 := geohash.EncodeWithPrecision(s.Latitude, s.Longitude, 1)
	p4 := geohash.EncodeWithPrecision(s.Latitude, s.Longitude, 4)
	p8 := geohash.EncodeWithPrecision(s.Latitude, s.Longitude, 8)
	shard := crc32.ChecksumIEEE([]byte(s.Id))%uint32(shardCount) + 1

	return types.StationItem{
		PK:        fmt.Sprintf("%d#%s", shard, p1),
		SK:        p8,
		GSI1PK:    p4,
		GSI1SK:    p8,
		StationId: s.Id,
		Title:     s.Title,
		Brand:     s.Brand,
		Address:   s.Address,
		Date:      s.Date,
		Latitude:  s.Latitude,
		Longitude: s.Longitude,
		Ulp91:     s.Price.Ulp91,
		Ulp95:     s.Price.Ulp95,
		Ulp98:     s.Price.Ulp98,
		Diesel:    s.Price.Diesel,
	}
}
