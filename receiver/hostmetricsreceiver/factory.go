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

package hostmetricsreceiver

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/cpuscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/diskscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/filesystemscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/loadscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/memoryscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/networkscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/processesscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/processscraper"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/scraper/swapscraper"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

// This file implements Factory for HostMetrics receiver.

const (
	// The value of "type" key in configuration.
	typeStr     = "hostmetrics"
	scrapersKey = "scrapers"
)

var (
	scraperFactories = map[string]internal.ScraperFactory{
		cpuscraper.TypeStr:        &cpuscraper.Factory{},
		diskscraper.TypeStr:       &diskscraper.Factory{},
		loadscraper.TypeStr:       &loadscraper.Factory{},
		filesystemscraper.TypeStr: &filesystemscraper.Factory{},
		memoryscraper.TypeStr:     &memoryscraper.Factory{},
		networkscraper.TypeStr:    &networkscraper.Factory{},
		processesscraper.TypeStr:  &processesscraper.Factory{},
		swapscraper.TypeStr:       &swapscraper.Factory{},
	}

	resourceScraperFactories = map[string]internal.ResourceScraperFactory{
		processscraper.TypeStr: &processscraper.Factory{},
	}
)

// NewFactory creates a new factory for host metrics receiver.
func NewFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		receiverhelper.WithMetrics(createMetricsReceiver),
		receiverhelper.WithCustomUnmarshaler(customUnmarshaler))
}

// customUnmarshaler returns custom unmarshaler for this config.
func customUnmarshaler(componentViperSection *viper.Viper, intoCfg interface{}) error {

	// load the non-dynamic config normally

	err := componentViperSection.Unmarshal(intoCfg)
	if err != nil {
		return err
	}

	cfg, ok := intoCfg.(*Config)
	if !ok {
		return fmt.Errorf("config type not hostmetrics.Config")
	}

	// dynamically load the individual collector configs based on the key name

	cfg.Scrapers = map[string]internal.Config{}

	scrapersViperSection, err := config.ViperSubExact(componentViperSection, scrapersKey)
	if err != nil {
		return err
	}
	if len(scrapersViperSection.AllKeys()) == 0 {
		return errors.New("must specify at least one scraper when using hostmetrics receiver")
	}

	for key := range componentViperSection.GetStringMap(scrapersKey) {
		factory, ok := getScraperFactory(key)
		if !ok {
			return fmt.Errorf("invalid scraper key: %s", key)
		}

		collectorCfg := factory.CreateDefaultConfig()
		collectorViperSection, err := config.ViperSubExact(scrapersViperSection, key)
		if err != nil {
			return err
		}
		err = collectorViperSection.UnmarshalExact(collectorCfg)
		if err != nil {
			return fmt.Errorf("error reading settings for scraper type %q: %v", key, err)
		}

		cfg.Scrapers[key] = collectorCfg
	}

	return nil
}

func getScraperFactory(key string) (internal.BaseFactory, bool) {
	if factory, ok := scraperFactories[key]; ok {
		return factory, true
	}

	if factory, ok := resourceScraperFactories[key]; ok {
		return factory, true
	}

	return nil, false
}

// createDefaultConfig creates the default configuration for receiver.
func createDefaultConfig() configmodels.Receiver {
	return &Config{ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(typeStr)}
}

// createMetricsReceiver creates a metrics receiver based on provided config.
func createMetricsReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (component.MetricsReceiver, error) {
	oCfg := cfg.(*Config)

	addScraperOptions, err := createAddScraperOptions(ctx, params.Logger, oCfg, scraperFactories, resourceScraperFactories)
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewScraperControllerReceiver(
		&oCfg.ScraperControllerSettings,
		params.Logger,
		consumer,
		addScraperOptions...,
	)
}

func createAddScraperOptions(
	ctx context.Context,
	logger *zap.Logger,
	config *Config,
	factories map[string]internal.ScraperFactory,
	resourceFactories map[string]internal.ResourceScraperFactory,
) ([]scraperhelper.ScraperControllerOption, error) {
	scraperControllerOptions := make([]scraperhelper.ScraperControllerOption, 0, len(config.Scrapers))

	for key, cfg := range config.Scrapers {
		hostMetricsScraper, ok, err := createHostMetricsScraper(ctx, logger, key, cfg, factories)
		if err != nil {
			return nil, fmt.Errorf("failed to create scraper for key %q: %w", key, err)
		}

		if ok {
			scraperControllerOptions = append(scraperControllerOptions, scraperhelper.AddMetricsScraper(hostMetricsScraper))
			continue
		}

		resourceMetricsScraper, ok, err := createResourceMetricsScraper(ctx, logger, key, cfg, resourceFactories)
		if err != nil {
			return nil, fmt.Errorf("failed to create resource scraper for key %q: %w", key, err)
		}

		if ok {
			scraperControllerOptions = append(scraperControllerOptions, scraperhelper.AddResourceMetricsScraper(resourceMetricsScraper))
			continue
		}

		return nil, fmt.Errorf("host metrics scraper factory not found for key: %q", key)
	}

	return scraperControllerOptions, nil
}

func createHostMetricsScraper(ctx context.Context, logger *zap.Logger, key string, cfg internal.Config, factories map[string]internal.ScraperFactory) (scraper scraperhelper.MetricsScraper, ok bool, err error) {
	factory := factories[key]
	if factory == nil {
		ok = false
		return
	}

	ok = true
	scraper, err = factory.CreateMetricsScraper(ctx, logger, cfg)
	return
}

func createResourceMetricsScraper(ctx context.Context, logger *zap.Logger, key string, cfg internal.Config, factories map[string]internal.ResourceScraperFactory) (scraper scraperhelper.ResourceMetricsScraper, ok bool, err error) {
	factory := factories[key]
	if factory == nil {
		ok = false
		return
	}

	ok = true
	scraper, err = factory.CreateResourceMetricsScraper(ctx, logger, cfg)
	return
}
