// Copyright  OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
)

// Scraper gathers metrics from the host machine and converts
// these into internal metrics format.
type Scraper interface {
	// Start performs any timely initialization tasks such as
	// setting up performance counters for initial collection,
	// and then begins scraping metrics at the configured
	// collection interval.
	Start(ctx context.Context) error
	// Close should clean up any unmanaged resources such as
	// performance counter handles.
	Close(ctx context.Context) error
}

// Factory can create a Scraper.
type Factory interface {
	// Type gets the type of the scraper created by this factory.
	Type() string

	// CreateDefaultConfig creates the default configuration for the Scraper.
	CreateDefaultConfig() Config

	// CreateMetricsScraper creates a scraper based on this config.
	// If the config is not valid, error will be returned instead.
	CreateMetricsScraper(ctx context.Context,
		logger *zap.Logger, cfg Config) (Scraper, error)
}

// Config is the configuration of a scraper.
type Config interface {
}
