package infoscraper

import (
	"context"

	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.uber.org/zap"
)

// This file implements Factory for Info scraper.

const (
	// The value of "type" key in configuration.
	TypeStr = "info"
)

// Factory is the Factory for scraper.
type Factory struct {
}

func (f *Factory) CreateDefaultConfig() internal.Config {
	return nil
}

// Type gets the type of the scraper config created by this Factory.
func (f *Factory) Type() string {
	return TypeStr
}

// CreateMetricsScraper creates a scraper based on provided config.
func (f *Factory) CreateMetricsScraper(
	ctx context.Context,
	_ *zap.Logger,
	cfg internal.Config,
) (scraperhelper.MetricsScraper, error) {
	s, err := newInfoScraper(ctx)
	if err != nil {
		return nil, err
	}

	ms := scraperhelper.NewMetricsScraper(TypeStr, s.Scrape)

	return ms, nil
}
