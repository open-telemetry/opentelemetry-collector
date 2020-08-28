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
	"fmt"
	"time"

	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

// receiver is the type that scrapes various host metrics.
type receiver struct {
	config *Config

	hostMetricScrapers     []internal.Scraper
	resourceMetricScrapers []internal.ResourceScraper

	consumer consumer.MetricsConsumer
	done     chan struct{}
}

// newHostMetricsReceiver creates a host metrics scraper.
func newHostMetricsReceiver(
	ctx context.Context,
	logger *zap.Logger,
	config *Config,
	factories map[string]internal.ScraperFactory,
	resourceFactories map[string]internal.ResourceScraperFactory,
	consumer consumer.MetricsConsumer,
) (*receiver, error) {

	hostMetricScrapers := make([]internal.Scraper, 0)
	resourceMetricScrapers := make([]internal.ResourceScraper, 0)

	for key, cfg := range config.Scrapers {
		hostMetricsScraper, ok, err := createHostMetricsScraper(ctx, logger, key, cfg, factories)
		if err != nil {
			return nil, fmt.Errorf("failed to create scraper for key %q: %w", key, err)
		}

		if ok {
			hostMetricScrapers = append(hostMetricScrapers, hostMetricsScraper)
			continue
		}

		resourceMetricsScraper, ok, err := createResourceMetricsScraper(ctx, logger, key, cfg, resourceFactories)
		if err != nil {
			return nil, fmt.Errorf("failed to create resource scraper for key %q: %w", key, err)
		}

		if ok {
			resourceMetricScrapers = append(resourceMetricScrapers, resourceMetricsScraper)
			continue
		}

		return nil, fmt.Errorf("host metrics scraper factory not found for key: %q", key)
	}

	hmr := &receiver{
		config:                 config,
		hostMetricScrapers:     hostMetricScrapers,
		resourceMetricScrapers: resourceMetricScrapers,
		consumer:               consumer,
	}

	return hmr, nil
}

func createHostMetricsScraper(ctx context.Context, logger *zap.Logger, key string, cfg internal.Config, factories map[string]internal.ScraperFactory) (scraper internal.Scraper, ok bool, err error) {
	factory := factories[key]
	if factory == nil {
		ok = false
		return
	}

	ok = true
	scraper, err = factory.CreateMetricsScraper(ctx, logger, cfg)
	return
}

func createResourceMetricsScraper(ctx context.Context, logger *zap.Logger, key string, cfg internal.Config, factories map[string]internal.ResourceScraperFactory) (scraper internal.ResourceScraper, ok bool, err error) {
	factory := factories[key]
	if factory == nil {
		ok = false
		return
	}

	ok = true
	scraper, err = factory.CreateMetricsScraper(ctx, logger, cfg)
	return
}

// Start initializes the underlying scrapers and begins scraping
// host metrics based on the OS platform.
func (hmr *receiver) Start(ctx context.Context, host component.Host) error {
	hmr.done = make(chan struct{})

	go func() {
		hmr.initializeScrapers(ctx, host)
		hmr.startScrapers()
	}()

	return nil
}

// Shutdown terminates all tickers and stops the underlying scrapers.
func (hmr *receiver) Shutdown(ctx context.Context) error {
	close(hmr.done)
	return hmr.closeScrapers(ctx)
}

func (hmr *receiver) initializeScrapers(ctx context.Context, host component.Host) {
	for _, scraper := range hmr.allScrapers() {
		err := scraper.Initialize(ctx)
		if err != nil {
			host.ReportFatalError(err)
			return
		}
	}
}

func (hmr *receiver) startScrapers() {
	go func() {
		ticker := time.NewTicker(hmr.config.CollectionInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				hmr.scrapeMetrics(context.Background())
			case <-hmr.done:
				return
			}
		}
	}()
}

func (hmr *receiver) scrapeMetrics(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "hostmetricsreceiver.ScrapeMetrics")
	defer span.End()

	var errors []error
	metricData := dataold.NewMetricData()

	if err := hmr.scrapeAndAppendHostMetrics(ctx, metricData); err != nil {
		errors = append(errors, err)
	}

	if err := hmr.scrapeAndAppendResourceMetrics(ctx, metricData); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Error(s) when scraping metrics: %v", componenterror.CombineErrors(errors))})
	}

	if err := hmr.consumer.ConsumeMetrics(ctx, pdatautil.MetricsFromOldInternalMetrics(metricData)); err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Unable to process metrics: %v", err)})
		return
	}
}

func (hmr *receiver) scrapeAndAppendHostMetrics(ctx context.Context, metricData dataold.MetricData) error {
	if len(hmr.hostMetricScrapers) == 0 {
		return nil
	}

	metrics := internal.InitializeMetricSlice(metricData)

	var errors []error
	for _, scraper := range hmr.hostMetricScrapers {
		scraperMetrics, err := scraper.ScrapeMetrics(ctx)
		if err != nil {
			errors = append(errors, err)
		}

		scraperMetrics.MoveAndAppendTo(metrics)
	}

	return componenterror.CombineErrors(errors)
}

func (hmr *receiver) scrapeAndAppendResourceMetrics(ctx context.Context, metricData dataold.MetricData) error {
	if len(hmr.resourceMetricScrapers) == 0 {
		return nil
	}

	rm := metricData.ResourceMetrics()

	var errors []error
	for _, scraper := range hmr.resourceMetricScrapers {
		scraperResourceMetrics, err := scraper.ScrapeMetrics(ctx)
		if err != nil {
			errors = append(errors, err)
		}

		scraperResourceMetrics.MoveAndAppendTo(rm)
	}

	return componenterror.CombineErrors(errors)
}

func (hmr *receiver) closeScrapers(ctx context.Context) error {
	var errs []error
	for _, scraper := range hmr.allScrapers() {
		err := scraper.Close(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return componenterror.CombineErrors(errs)
	}

	return nil
}

func (hmr *receiver) allScrapers() []internal.BaseScraper {
	allScrapers := make([]internal.BaseScraper, len(hmr.hostMetricScrapers)+len(hmr.resourceMetricScrapers))
	for i, hostMetricScraper := range hmr.hostMetricScrapers {
		allScrapers[i] = hostMetricScraper
	}
	startIdx := len(hmr.hostMetricScrapers)
	for i, resourceMetricScraper := range hmr.resourceMetricScrapers {
		allScrapers[startIdx+i] = resourceMetricScraper
	}
	return allScrapers
}
