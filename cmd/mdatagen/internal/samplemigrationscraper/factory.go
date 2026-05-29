// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplemigrationscraper // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/samplemigrationscraper"

import (
	"context"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/samplemigrationscraper/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/xscraper"
)

// NewFactory returns a receiver.Factory for sample receiver.
func NewFactory() scraper.Factory {
	return xscraper.NewFactory(
		metadata.Type,
		createDefaultConfig,
		xscraper.WithMetrics(createMetrics, metadata.MetricsStability),
		xscraper.WithLogs(createLogs, metadata.LogsStability),
	)
}

func createMetrics(context.Context, scraper.Settings, component.Config) (scraper.Metrics, error) {
	return scraper.NewMetrics(func(context.Context) (pmetric.Metrics, error) {
		return pmetric.NewMetrics(), nil
	})
}

func createLogs(context.Context, scraper.Settings, component.Config) (scraper.Logs, error) {
	return scraper.NewLogs(func(context.Context) (plog.Logs, error) {
		return plog.NewLogs(), nil
	})
}
