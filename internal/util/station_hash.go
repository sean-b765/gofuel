package util

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
)

// Create an id for deduping station data
func Md5Hash(address string, lat float64, lng float64) string {
	dataString := fmt.Sprintf("%s|%f|%f", address, lat, lng)
	hashArray := md5.Sum([]byte(dataString))
	return hex.EncodeToString(hashArray[:])
}
