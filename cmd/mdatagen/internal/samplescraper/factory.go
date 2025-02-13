// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplescraper // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplescraper"

import (
	"context"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/samplescraper/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/scraper"
)

// NewFactory returns a receiver.Factory for sample receiver.
func NewFactory() scraper.Factory {
	return scraper.NewFactory(
		metadata.Type,
		func() component.Config { return &struct{}{} },
		scraper.WithMetrics(createMetrics, metadata.MetricsStability))
}

func createMetrics(context.Context, scraper.Settings, component.Config) (scraper.Metrics, error) {
	return scraper.NewMetrics(func(context.Context) (pmetric.Metrics, error) {
		return pmetric.NewMetrics(), nil
	})
}
