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
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
	"go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"
)

type ControllerConfig = controller.ControllerConfig

// ScheduleConfig defines the interval and timing for one collection schedule.
// It is used with WithMetricsSchedules or WithLogsSchedules to collect different
// metrics or events at different intervals (e.g. per-metric or per-event collection_interval).
type ScheduleConfig = controller.ScheduleConfig

// MetricsScraperController is the interface passed to each schedule's ScrapeFunc when using
// NewMetricsControllerWithSchedules. Receivers can use it to access scrapers and observability.
type MetricsScraperController interface {
	Scrapers() []scraper.Metrics
	Timeout() time.Duration
	Obsrecv() *receiverhelper.ObsReport
}

// LogsScraperController is the interface passed to each schedule's ScrapeFunc when using
// NewLogsControllerWithSchedules. Receivers can use it to access scrapers and observability.
type LogsScraperController interface {
	Scrapers() []scraper.Logs
	Timeout() time.Duration
	Obsrecv() *receiverhelper.ObsReport
}

// MetricsSchedule defines one collection schedule for metrics (interval + scrape function).
type MetricsSchedule struct {
	Config     ScheduleConfig
	ScrapeFunc func(MetricsScraperController)
}

// LogsSchedule defines one collection schedule for logs (interval + scrape function).
type LogsSchedule struct {
	Config     ScheduleConfig
	ScrapeFunc func(LogsScraperController)
}

// NewLogsScheduleForScraper returns a LogsSchedule that scrapes a single scraper by index and
// forwards logs to the consumer with observability. Use this with WithLogsSchedules when each
// schedule should run one scraper at its own interval (e.g. per-event collection_interval).
// This avoids receivers implementing the full ScrapeFunc boilerplate (context, scrape, obsrecv, consume).
func NewLogsScheduleForScraper(scraperIndex int, cfg ScheduleConfig, nextConsumer consumer.Logs) LogsSchedule {
	return LogsSchedule{
		Config: cfg,
		ScrapeFunc: func(c LogsScraperController) {
			ctx, cancel := WithScrapeContext(cfg.Timeout)
			defer cancel()
			logs, err := c.Scrapers()[scraperIndex].ScrapeLogs(ctx)
			if err != nil && !scrapererror.IsPartialScrapeError(err) {
				return
			}
			count := logs.LogRecordCount()
			ctx = c.Obsrecv().StartMetricsOp(ctx)
			consumeErr := nextConsumer.ConsumeLogs(ctx, logs)
			c.Obsrecv().EndMetricsOp(ctx, "", count, consumeErr)
		},
	}
}

// NewMetricsScheduleForScraper returns a MetricsSchedule that scrapes a single scraper by index and
// forwards metrics to the consumer with observability. Use this with WithMetricsSchedules when each
// schedule should run one scraper at its own interval (e.g. per-metric collection_interval).
// This avoids receivers implementing the full ScrapeFunc boilerplate (context, scrape, obsrecv, consume).
func NewMetricsScheduleForScraper(scraperIndex int, cfg ScheduleConfig, nextConsumer consumer.Metrics) MetricsSchedule {
	return MetricsSchedule{
		Config: cfg,
		ScrapeFunc: func(c MetricsScraperController) {
			ctx, cancel := WithScrapeContext(cfg.Timeout)
			defer cancel()
			metrics, err := c.Scrapers()[scraperIndex].ScrapeMetrics(ctx)
			if err != nil && !scrapererror.IsPartialScrapeError(err) {
				return
			}
			dataPointCount := metrics.DataPointCount()
			ctx = c.Obsrecv().StartMetricsOp(ctx)
			consumeErr := nextConsumer.ConsumeMetrics(ctx, metrics)
			c.Obsrecv().EndMetricsOp(ctx, "", dataPointCount, consumeErr)
		},
	}
}

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

// WithMetricsSchedules configures multiple collection schedules for metrics, each with its own
// interval and scrape function. When set, NewMetricsController uses these schedules instead
// of a single collection_interval. Use this to collect different metrics or events at different
// intervals (e.g. per-metric or per-event collection_interval in config).
func WithMetricsSchedules(schedules []MetricsSchedule) ControllerOption {
	return optionFunc(func(o *controllerOptions) {
		o.metricsSchedules = schedules
	})
}

// WithLogsSchedules configures multiple collection schedules for logs, each with its own
// interval and scrape function. When set, NewLogsController uses these schedules instead
// of a single collection_interval.
func WithLogsSchedules(schedules []LogsSchedule) ControllerOption {
	return optionFunc(func(o *controllerOptions) {
		o.logsSchedules = schedules
	})
}

type factoryWithConfig struct {
	f   scraper.Factory
	cfg component.Config
}

type controllerOptions struct {
	tickerCh             <-chan time.Time
	factoriesWithConfig   []factoryWithConfig
	metricsSchedules      []MetricsSchedule
	logsSchedules         []LogsSchedule
}

// NewLogsController creates a receiver.Logs with the configured options, that can control multiple scraper.Logs.
// When WithLogsSchedules is used, each schedule runs with its own collection interval and scrape function.
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
	if len(co.logsSchedules) > 0 {
		internalSchedules := make([]controller.Schedule[scraper.Logs], len(co.logsSchedules))
		for i, s := range co.logsSchedules {
			s := s
			internalSchedules[i] = controller.Schedule[scraper.Logs]{
				Config:     s.Config,
				ScrapeFunc: func(c *controller.Controller[scraper.Logs]) { s.ScrapeFunc(c) },
			}
		}
		return controller.NewControllerWithSchedules(rSet, scrapers, internalSchedules)
	}
	return controller.NewController[scraper.Logs](
		cfg, rSet, scrapers, func(c *controller.Controller[scraper.Logs]) { scrapeLogs(c, nextConsumer) }, co.tickerCh)
}

// NewMetricsController creates a receiver.Metrics with the configured options, that can control multiple scraper.Metrics.
// When WithMetricsSchedules is used, each schedule runs with its own collection interval and scrape function.
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
	if len(co.metricsSchedules) > 0 {
		internalSchedules := make([]controller.Schedule[scraper.Metrics], len(co.metricsSchedules))
		for i, s := range co.metricsSchedules {
			s := s
			internalSchedules[i] = controller.Schedule[scraper.Metrics]{
				Config:     s.Config,
				ScrapeFunc: func(c *controller.Controller[scraper.Metrics]) { s.ScrapeFunc(c) },
			}
		}
		return controller.NewControllerWithSchedules(rSet, scrapers, internalSchedules)
	}
	return controller.NewController[scraper.Metrics](
		cfg, rSet, scrapers, func(c *controller.Controller[scraper.Metrics]) { scrapeMetrics(c, nextConsumer) }, co.tickerCh)
}

func scrapeLogs(c *controller.Controller[scraper.Logs], nextConsumer consumer.Logs) {
	ctx, done := controller.WithScrapeContext(c.Timeout())
	defer done()

	logs := plog.NewLogs()
	for i := range c.Scrapers() {
		md, err := c.Scrapers()[i].ScrapeLogs(ctx)
		if err != nil && !scrapererror.IsPartialScrapeError(err) {
			continue
		}
		md.ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
	}

	logRecordCount := logs.LogRecordCount()
	ctx = c.Obsrecv().StartMetricsOp(ctx)
	err := nextConsumer.ConsumeLogs(ctx, logs)
	c.Obsrecv().EndMetricsOp(ctx, "", logRecordCount, err)
}

func scrapeMetrics(c *controller.Controller[scraper.Metrics], nextConsumer consumer.Metrics) {
	ctx, done := controller.WithScrapeContext(c.Timeout())
	defer done()

	metrics := pmetric.NewMetrics()
	for i := range c.Scrapers() {
		md, err := c.Scrapers()[i].ScrapeMetrics(ctx)
		if err != nil && !scrapererror.IsPartialScrapeError(err) {
			continue
		}
		md.ResourceMetrics().MoveAndAppendTo(metrics.ResourceMetrics())
	}

	dataPointCount := metrics.DataPointCount()
	ctx = c.Obsrecv().StartMetricsOp(ctx)
	err := nextConsumer.ConsumeMetrics(ctx, metrics)
	c.Obsrecv().EndMetricsOp(ctx, "", dataPointCount, err)
}

// WithScrapeContext returns a context with an optional deadline for scrape operations.
// If timeout is 0, the context has no deadline. Use this in custom schedule ScrapeFuncs
// with the schedule's Config.Timeout for per-schedule timeouts.
func WithScrapeContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return controller.WithScrapeContext(timeout)
}

func getOptions(options []ControllerOption) controllerOptions {
	co := controllerOptions{}
	for _, op := range options {
		op.apply(&co)
	}
	return co
}
