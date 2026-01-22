// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/scraper/scraperhelper"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
	"go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"
)

type ControllerConfig = controller.ControllerConfig

// NewDefaultControllerConfig returns default scraper controller
// settings with a collection interval of one minute.
func NewDefaultControllerConfig() ControllerConfig {
	return controller.NewDefaultControllerConfig()
}

// ControllerOption apply changes to internal options.
type ControllerOption interface {
	apply(*controllerOptions)
}

type optionFunc func(*controllerOptions)

func (of optionFunc) apply(e *controllerOptions) {
	of(e)
}

// AddMetricsScraper configures the scraper.Metrics to be called with the
// specified options, and at the specified collection interval.
//
// Observability information will be reported, and the scraped metrics
// will be passed to the next consumer.
func AddMetricsScraper(t component.Type, sc scraper.Metrics) ControllerOption {
	f := scraper.NewFactory(t, nil,
		scraper.WithMetrics(func(context.Context, scraper.Settings, component.Config) (scraper.Metrics, error) {
			return sc, nil
		}, component.StabilityLevelAlpha))
	return AddFactoryWithConfig(f, nil)
}

// AddScraper configures the scraper.Metrics to be called with the
// specified options, and at the specified collection interval.
//
// Observability information will be reported, and the scraped metrics
// will be passed to the next consumer.
//
// Deprecated: [0.144.0] Use AddMetricsScraper instead.
func AddScraper(t component.Type, sc scraper.Metrics) ControllerOption {
	return AddMetricsScraper(t, sc)
}

// AddFactoryWithConfig configures the scraper.Factory and associated config that
// will be used to create a new scraper. The created scraper will be called with
// the specified options, and at the specified collection interval.
//
// Observability information will be reported, and the scraped metrics
// will be passed to the next consumer.
func AddFactoryWithConfig(f scraper.Factory, cfg component.Config) ControllerOption {
	return optionFunc(func(o *controllerOptions) {
		o.factoriesWithConfig = append(o.factoriesWithConfig, factoryWithConfig{f: f, cfg: cfg})
	})
}

// WithTickerChannel allows you to override the scraper controller's ticker
// channel to specify when scrape is called. This is only expected to be
// used by tests.
func WithTickerChannel(tickerCh <-chan time.Time) ControllerOption {
	return optionFunc(func(o *controllerOptions) {
		o.tickerCh = tickerCh
	})
}

type factoryWithConfig struct {
	f   scraper.Factory
	cfg component.Config
}

type controllerOptions struct {
	tickerCh            <-chan time.Time
	factoriesWithConfig []factoryWithConfig
}

// NewLogsController creates a receiver.Logs with the configured options, that can control multiple scraper.Logs.
func NewLogsController(cfg *ControllerConfig,
	rSet receiver.Settings,
	nextConsumer consumer.Logs,
	options ...ControllerOption,
) (receiver.Logs, error) {
	co := getOptions(options)
	scrapers := make([]scraper.Logs, 0, len(co.factoriesWithConfig))
	for _, fwc := range co.factoriesWithConfig {
		set := controller.GetSettings(fwc.f.Type(), rSet)
		s, err := fwc.f.CreateLogs(context.Background(), set, fwc.cfg)
		if err != nil {
			return nil, err
		}
		s, err = wrapObsLogs(s, rSet.ID, set.ID, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		scrapers = append(scrapers, s)
	}
	return controller.NewController[scraper.Logs](
		cfg, rSet, scrapers, func(c *controller.Controller[scraper.Logs]) { scrapeLogs(c, nextConsumer) }, co.tickerCh)
}

// NewMetricsController creates a receiver.Metrics with the configured options, that can control multiple scraper.Metrics.
func NewMetricsController(cfg *ControllerConfig,
	rSet receiver.Settings,
	nextConsumer consumer.Metrics,
	options ...ControllerOption,
) (receiver.Metrics, error) {
	co := getOptions(options)
	scrapers := make([]scraper.Metrics, 0, len(co.factoriesWithConfig))
	for _, fwc := range co.factoriesWithConfig {
		set := controller.GetSettings(fwc.f.Type(), rSet)
		s, err := fwc.f.CreateMetrics(context.Background(), set, fwc.cfg)
		if err != nil {
			return nil, err
		}
		s, err = wrapObsMetrics(s, rSet.ID, set.ID, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		scrapers = append(scrapers, s)
	}
	return controller.NewController[scraper.Metrics](
		cfg, rSet, scrapers, func(c *controller.Controller[scraper.Metrics]) { scrapeMetrics(c, nextConsumer) }, co.tickerCh)
}

func scrapeLogs(c *controller.Controller[scraper.Logs], nextConsumer consumer.Logs) {
	ctx, done := controller.WithScrapeContext(c.Timeout)
	defer done()

	logs := plog.NewLogs()
	for i := range c.Scrapers {
		md, err := c.Scrapers[i].ScrapeLogs(ctx)
		if err != nil && !scrapererror.IsPartialScrapeError(err) {
			continue
		}
		md.ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
	}

	logRecordCount := logs.LogRecordCount()
	ctx = c.Obsrecv.StartMetricsOp(ctx)
	err := nextConsumer.ConsumeLogs(ctx, logs)
	c.Obsrecv.EndMetricsOp(ctx, "", logRecordCount, err)
}

func scrapeMetrics(c *controller.Controller[scraper.Metrics], nextConsumer consumer.Metrics) {
	ctx, done := controller.WithScrapeContext(c.Timeout)
	defer done()

	metrics := pmetric.NewMetrics()
	for i := range c.Scrapers {
		md, err := c.Scrapers[i].ScrapeMetrics(ctx)
		if err != nil && !scrapererror.IsPartialScrapeError(err) {
			continue
		}
		md.ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
	}

	dataPointCount := metrics.DataPointCount()
	ctx = c.Obsrecv.StartMetricsOp(ctx)
	err := nextConsumer.ConsumeMetrics(ctx, metrics)
	c.Obsrecv.EndMetricsOp(ctx, "", dataPointCount, err)
}

func getOptions(options []ControllerOption) controllerOptions {
	co := controllerOptions{}
	for _, op := range options {
		op.apply(&co)
	}
	return co
}
