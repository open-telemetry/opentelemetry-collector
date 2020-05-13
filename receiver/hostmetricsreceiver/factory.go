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
	"time"

	"github.com/spf13/viper"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/cpuscraper"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/diskscraper"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/filesystemscraper"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/memoryscraper"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/scraper/networkscraper"
)

// This file implements Factory for HostMetrics receiver.

const (
	// The value of "type" key in configuration.
	typeStr     = "hostmetrics"
	scrapersKey = "scrapers"
)

// Factory is the Factory for receiver.
type Factory struct {
	scraperFactories map[string]internal.Factory
}

// NewFactory creates a new factory for host metrics receiver.
func NewFactory() *Factory {
	return &Factory{
		scraperFactories: map[string]internal.Factory{
			cpuscraper.TypeStr:        &cpuscraper.Factory{},
			diskscraper.TypeStr:       &diskscraper.Factory{},
			filesystemscraper.TypeStr: &filesystemscraper.Factory{},
			memoryscraper.TypeStr:     &memoryscraper.Factory{},
			networkscraper.TypeStr:    &networkscraper.Factory{},
		},
	}
}

// Type returns the type of the Receiver config created by this Factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CustomUnmarshaler returns custom unmarshaler for this config.
func (f *Factory) CustomUnmarshaler() component.CustomUnmarshaler {
	return func(componentViperSection *viper.Viper, intoCfg interface{}) error {

		// load the non-dynamic config normally

		err := componentViperSection.Unmarshal(intoCfg)
		if err != nil {
			return err
		}

		cfg, ok := intoCfg.(*Config)
		if !ok {
			return fmt.Errorf("config type not hostmetrics.Config")
		}

		if cfg.CollectionInterval <= 0 {
			return fmt.Errorf("collection_interval must be a positive duration")
		}

		// dynamically load the individual collector configs based on the key name

		cfg.Scrapers = map[string]internal.Config{}

		scrapersViperSection := config.ViperSub(componentViperSection, scrapersKey)
		if scrapersViperSection == nil || len(scrapersViperSection.AllKeys()) == 0 {
			return fmt.Errorf("must specify at least one scraper when using hostmetrics receiver")
		}

		for key := range componentViperSection.GetStringMap(scrapersKey) {
			factory, ok := f.scraperFactories[key]
			if !ok {
				return fmt.Errorf("invalid scraper key: %s", key)
			}

			collectorCfg := factory.CreateDefaultConfig()
			collectorViperSection := config.ViperSub(scrapersViperSection, key)
			err := collectorViperSection.UnmarshalExact(collectorCfg)
			if err != nil {
				return fmt.Errorf("error reading settings for scraper type %q: %v", key, err)
			}

			cfg.Scrapers[key] = collectorCfg
		}

		return nil
	}
}

// CreateDefaultConfig creates the default configuration for receiver.
func (f *Factory) CreateDefaultConfig() configmodels.Receiver {
	return &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		CollectionInterval: time.Minute,
	}
}

// CreateTraceReceiver returns error as trace receiver is not applicable to host metrics receiver.
func (f *Factory) CreateTraceReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	consumer consumer.TraceConsumer,
) (component.TraceReceiver, error) {
	// Host Metrics does not support traces
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func (f *Factory) CreateMetricsReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (component.MetricsReceiver, error) {
	config := cfg.(*Config)

	hmr, err := newHostMetricsReceiver(ctx, params.Logger, config, f.scraperFactories, consumer)
	if err != nil {
		return nil, err
	}

	return hmr, nil
}
