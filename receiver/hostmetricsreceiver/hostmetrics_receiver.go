// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hostmetricsreceiver

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
)

// Receiver is the type that scrapes various host metrics.
type Receiver struct {
	config   *Config
	scrapers []internal.Scraper
}

// NewHostMetricsReceiver creates a host metrics scraper.
func NewHostMetricsReceiver(
	ctx context.Context,
	logger *zap.Logger,
	config *Config,
	factories map[string]internal.Factory,
	consumer consumer.MetricsConsumer,
) (*Receiver, error) {

	scrapers := make([]internal.Scraper, 0)
	for key, cfg := range config.Scrapers {
		factory := factories[key]
		if factory == nil {
			return nil, fmt.Errorf("host metrics scraper factory not found for key: %s", key)
		}

		scraper, err := factory.CreateMetricsScraper(ctx, logger, cfg, consumer)
		if err != nil {
			return nil, fmt.Errorf("cannot create scraper: %s", err.Error())
		}
		scrapers = append(scrapers, scraper)
	}

	hmr := &Receiver{
		config:   config,
		scrapers: scrapers,
	}

	return hmr, nil
}

// Start begins scraping host metrics based on the OS platform.
func (hmr *Receiver) Start(ctx context.Context, host component.Host) error {
	go func() {
		for _, scraper := range hmr.scrapers {
			err := scraper.Start(ctx)
			if err != nil {
				host.ReportFatalError(err)
				return
			}
		}
	}()

	return nil
}

// Shutdown stops the underlying host metrics scrapers.
func (hmr *Receiver) Shutdown(ctx context.Context) error {
	var errs []error

	for _, scraper := range hmr.scrapers {
		err := scraper.Shutdown(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return componenterror.CombineErrors(errs)
	}

	return nil
}
