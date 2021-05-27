// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

// BaseFactory for creating Scrapers.
type BaseFactory interface {
	// CreateDefaultConfig creates the default configuration for the Scraper.
	CreateDefaultConfig() Config
}

// ScraperFactory can create a MetricScraper.
type ScraperFactory interface {
	BaseFactory

	// CreateMetricsScraper creates a scraper based on this config.
	// If the config is not valid, error will be returned instead.
	CreateMetricsScraper(ctx context.Context, logger *zap.Logger, cfg Config) (scraperhelper.MetricsScraper, error)
}

// ResourceScraperFactory can create a ResourceScraper.
type ResourceScraperFactory interface {
	BaseFactory

	// CreateResourceMetricsScraper creates a resource scraper based on this
	// config. If the config is not valid, error will be returned instead.
	CreateResourceMetricsScraper(ctx context.Context, logger *zap.Logger, cfg Config) (scraperhelper.ResourceMetricsScraper, error)
}

// Config is the configuration of a scraper.
type Config interface {
}

// ConfigSettings provides common settings for scraper configuration.
type ConfigSettings struct {
}
