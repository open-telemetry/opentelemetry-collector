package infoscraper

import "go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"

// Config relating to Load Metric Scraper.
type Config struct {
	internal.ConfigSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}
