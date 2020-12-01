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

package processscraper

import (
	"context"
	"errors"
	"runtime"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

// This file implements Factory for Process scraper.

const (
	// The value of "type" key in configuration.
	TypeStr = "process"
)

// Factory is the Factory for scraper.
type Factory struct {
}

// CreateDefaultConfig creates the default configuration for the Scraper.
func (f *Factory) CreateDefaultConfig() internal.Config {
	return &Config{}
}

// CreateResourceMetricsScraper creates a resource scraper based on provided config.
func (f *Factory) CreateResourceMetricsScraper(
	_ context.Context,
	_ *zap.Logger,
	config internal.Config,
) (scraperhelper.ResourceMetricsScraper, error) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		return nil, errors.New("process scraper only available on Linux or Windows")
	}

	cfg := config.(*Config)
	s, err := newProcessScraper(cfg)
	if err != nil {
		return nil, err
	}

	ms := scraperhelper.NewResourceMetricsScraper(
		TypeStr,
		s.scrape,
		scraperhelper.WithStart(s.start),
	)

	return ms, nil
}
