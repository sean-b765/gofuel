package providers

import (
	"sync"

	"seanboaden.dev/fuel/internal/types"
)

func FetchAllStations() []types.Station {
	var wg sync.WaitGroup
	var wa, nswTas, saQld []types.Station

	wg.Add(3)
	go func() { defer wg.Done(); wa = GetWaPrices("") }()
	go func() { defer wg.Done(); nswTas = GetNswTasPricesCurrent() }()
	go func() { defer wg.Done(); saQld = GetSaQldPricesCurrent() }()
	wg.Wait()

	items := make([]types.Station, 0, len(wa)+len(nswTas)+len(saQld))
	items = append(items, wa...)
	items = append(items, nswTas...)
	items = append(items, saQld...)
	return items
}

// FetchProviderAndDay fetches a single provider. day is "" or "tomorrow"
// (WA only; ignored by other providers).
func FetchProviderAndDay(provider, day string) []types.Station {
	switch provider {
	case "wa":
		return GetWaPrices(day)
	case "nsw_tas":
		return GetNswTasPricesCurrent()
	case "sa_qld":
		return GetSaQldPricesCurrent()
	default:
		return nil
	}
}
