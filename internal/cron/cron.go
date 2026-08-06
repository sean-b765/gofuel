package cron

import (
	"context"
	"fmt"
	"log"

	"seanboaden.dev/fuel/internal/providers"
	"seanboaden.dev/fuel/internal/store"
)

type Event struct {
	Provider string `json:"provider"`
	Day      string `json:"day"`
}

func Run(ctx context.Context, evt Event) error {
	if evt.Provider != "wa" && evt.Provider != "nsw_tas" && evt.Provider != "sa_qld" {
		return fmt.Errorf("invalid provider %q", evt.Provider)
	}
	if evt.Day != "" && evt.Day != "tomorrow" {
		return fmt.Errorf("invalid day %q", evt.Day)
	}

	stations := providers.FetchProviderAndDay(evt.Provider, evt.Day)
	log.Printf("[cron] provider=%s day=%s fetched=%d", evt.Provider, evt.Day, len(stations))

	if err := store.PutStations(stations); err != nil {
		return fmt.Errorf("write stations: %w", err)
	}
	return nil
}
