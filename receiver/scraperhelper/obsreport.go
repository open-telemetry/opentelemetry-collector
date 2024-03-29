// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

var (
	scraperScope = obsmetrics.Scope + obsmetrics.NameSep + obsmetrics.ScraperKey
)

// ObsReport is a helper to add observability to a scraper.
type ObsReport struct {
	level      configtelemetry.Level
	receiverID component.ID
	scraper    component.ID
	tracer     trace.Tracer

	logger *zap.Logger

	otelAttrs            []attribute.KeyValue
	scrapedMetricsPoints metric.Int64Counter
	erroredMetricsPoints metric.Int64Counter
}

// ObsReportSettings are settings for creating an ObsReport.
type ObsReportSettings struct {
	ReceiverID             component.ID
	Scraper                component.ID
	ReceiverCreateSettings receiver.CreateSettings
}

// NewObsReport creates a new ObsReport.
func NewObsReport(cfg ObsReportSettings) (*ObsReport, error) {
	return newScraper(cfg)
}

func newScraper(cfg ObsReportSettings) (*ObsReport, error) {
	scraper := &ObsReport{
		level:      cfg.ReceiverCreateSettings.TelemetrySettings.MetricsLevel,
		receiverID: cfg.ReceiverID,
		scraper:    cfg.Scraper,
		tracer:     cfg.ReceiverCreateSettings.TracerProvider.Tracer(cfg.Scraper.String()),

		logger: cfg.ReceiverCreateSettings.Logger,
		otelAttrs: []attribute.KeyValue{
			attribute.String(obsmetrics.ReceiverKey, cfg.ReceiverID.String()),
			attribute.String(obsmetrics.ScraperKey, cfg.Scraper.String()),
		},
	}

	if err := scraper.createOtelMetrics(cfg); err != nil {
		return nil, err
	}

	return scraper, nil
}

func (s *ObsReport) createOtelMetrics(cfg ObsReportSettings) error {
	meter := cfg.ReceiverCreateSettings.MeterProvider.Meter(scraperScope)

	var errors, err error

	s.scrapedMetricsPoints, err = meter.Int64Counter(
		obsmetrics.ScraperPrefix+obsmetrics.ScrapedMetricPointsKey,
		metric.WithDescription("Number of metric points successfully scraped."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	s.erroredMetricsPoints, err = meter.Int64Counter(
		obsmetrics.ScraperPrefix+obsmetrics.ErroredMetricPointsKey,
		metric.WithDescription("Number of metric points that were unable to be scraped."),
		metric.WithUnit("1"),
	)
	errors = multierr.Append(errors, err)

	return errors
}

// StartMetricsOp is called when a scrape operation is started. The
// returned context should be used in other calls to the obsreport functions
// dealing with the same scrape operation.
func (s *ObsReport) StartMetricsOp(ctx context.Context) context.Context {
	spanName := obsmetrics.ScraperPrefix + s.receiverID.String() + obsmetrics.NameSep + s.scraper.String() + obsmetrics.ScraperMetricsOperationSuffix
	ctx, _ = s.tracer.Start(ctx, spanName)
	return ctx
}

// EndMetricsOp completes the scrape operation that was started with
// StartMetricsOp.
func (s *ObsReport) EndMetricsOp(
	scraperCtx context.Context,
	numScrapedMetrics int,
	err error,
) {
	numErroredMetrics := 0
	if err != nil {
		var partialErr scrapererror.PartialScrapeError
		if errors.As(err, &partialErr) {
			numErroredMetrics = partialErr.Failed
		} else {
			numErroredMetrics = numScrapedMetrics
			numScrapedMetrics = 0
		}
	}

	span := trace.SpanFromContext(scraperCtx)

	if s.level != configtelemetry.LevelNone {
		s.recordMetrics(scraperCtx, numScrapedMetrics, numErroredMetrics)
	}

	// end span according to errors
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String(obsmetrics.FormatKey, component.DataTypeMetrics.String()),
			attribute.Int64(obsmetrics.ScrapedMetricPointsKey, int64(numScrapedMetrics)),
			attribute.Int64(obsmetrics.ErroredMetricPointsKey, int64(numErroredMetrics)),
		)

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
	}

	span.End()
}

func (s *ObsReport) recordMetrics(scraperCtx context.Context, numScrapedMetrics, numErroredMetrics int) {
	s.scrapedMetricsPoints.Add(scraperCtx, int64(numScrapedMetrics), metric.WithAttributes(s.otelAttrs...))
	s.erroredMetricsPoints.Add(scraperCtx, int64(numErroredMetrics), metric.WithAttributes(s.otelAttrs...))
}
