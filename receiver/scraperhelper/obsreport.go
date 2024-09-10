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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scrapererror"
	"go.opentelemetry.io/collector/receiver/scraperhelper/internal/metadata"
)

// obsReport is a helper to add observability to a scraper.
type obsReport struct {
	receiverID component.ID
	scraper    component.ID
	tracer     trace.Tracer

	otelAttrs        []attribute.KeyValue
	telemetryBuilder *metadata.TelemetryBuilder
}

// obsReportSettings are settings for creating an ObsReport.
type obsReportSettings struct {
	ReceiverID             component.ID
	Scraper                component.ID
	ReceiverCreateSettings receiver.Settings
}

func newScraper(cfg obsReportSettings) (*obsReport, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(cfg.ReceiverCreateSettings.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return &obsReport{
		receiverID: cfg.ReceiverID,
		scraper:    cfg.Scraper,
		tracer:     cfg.ReceiverCreateSettings.TracerProvider.Tracer(cfg.Scraper.String()),

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
func (s *obsReport) StartMetricsOp(ctx context.Context) context.Context {
	spanName := obsmetrics.ScraperPrefix + s.receiverID.String() + obsmetrics.SpanNameSep + s.scraper.String() + obsmetrics.ScraperMetricsOperationSuffix
	ctx, _ = s.tracer.Start(ctx, spanName)
	return ctx
}

// EndMetricsOp completes the scrape operation that was started with
// StartMetricsOp.
func (s *obsReport) EndMetricsOp(
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

	s.recordMetrics(scraperCtx, numScrapedMetrics, numErroredMetrics)

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

func (s *obsReport) recordMetrics(scraperCtx context.Context, numScrapedMetrics, numErroredMetrics int) {
	s.telemetryBuilder.ScraperScrapedMetricPoints.Add(scraperCtx, int64(numScrapedMetrics), metric.WithAttributes(s.otelAttrs...))
	s.telemetryBuilder.ScraperErroredMetricPoints.Add(scraperCtx, int64(numErroredMetrics), metric.WithAttributes(s.otelAttrs...))
}
