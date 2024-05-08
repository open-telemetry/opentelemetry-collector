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
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scrapererror"
	"go.opentelemetry.io/collector/receiver/scraperhelper/internal/metadata"
)

// ObsReport is a helper to add observability to a scraper.
type ObsReport struct {
	level      configtelemetry.Level
	receiverID component.ID
	scraper    component.ID
	tracer     trace.Tracer

	logger *zap.Logger

	otelAttrs        []attribute.KeyValue
	telemetryBuilder *metadata.TelemetryBuilder
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
	telemetryBuilder, err := metadata.NewTelemetryBuilder(cfg.ReceiverCreateSettings.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return &ObsReport{
		level:      cfg.ReceiverCreateSettings.TelemetrySettings.MetricsLevel,
		receiverID: cfg.ReceiverID,
		scraper:    cfg.Scraper,
		tracer:     cfg.ReceiverCreateSettings.TracerProvider.Tracer(cfg.Scraper.String()),

		logger: cfg.ReceiverCreateSettings.Logger,
		otelAttrs: []attribute.KeyValue{
			attribute.String(obsmetrics.ReceiverKey, cfg.ReceiverID.String()),
			attribute.String(obsmetrics.ScraperKey, cfg.Scraper.String()),
		},
		telemetryBuilder: telemetryBuilder,
	}, nil
}

// StartMetricsOp is called when a scrape operation is started. The
// returned context should be used in other calls to the obsreport functions
// dealing with the same scrape operation.
func (s *ObsReport) StartMetricsOp(ctx context.Context) context.Context {
	spanName := obsmetrics.ScraperPrefix + s.receiverID.String() + obsmetrics.SpanNameSep + s.scraper.String() + obsmetrics.ScraperMetricsOperationSuffix
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
	s.telemetryBuilder.ScraperScrapedMetricPoints.Add(scraperCtx, int64(numScrapedMetrics), metric.WithAttributes(s.otelAttrs...))
	s.telemetryBuilder.ScraperErroredMetricPoints.Add(scraperCtx, int64(numErroredMetrics), metric.WithAttributes(s.otelAttrs...))
}
