// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/scraper/scraperhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
	"go.opentelemetry.io/collector/scraper/scraperhelper/internal/metadata"
)

const (
	// scraperKey used to identify scrapers in metrics and traces.
	scraperKey = "scraper"
	// scrapedMetricPointsKey used to identify metric points scraped by the
	// Collector.
	scrapedMetricPointsKey = "scraped_metric_points"
	// erroredMetricPointsKey used to identify metric points errored (i.e.
	// unable to be scraped) by the Collector.
	erroredMetricPointsKey = "errored_metric_points"

	spanNameSep = "/"
	// receiverKey used to identify receivers in metrics and traces.
	receiverKey = "receiver"
	// FormatKey used to identify the format of the data received.
	formatKey = "format"
)

func newObsMetrics(delegate scraper.ScrapeMetricsFunc, receiverID component.ID, scraperID component.ID, telSettings component.TelemetrySettings) (scraper.ScrapeMetricsFunc, error) {
	telemetryBuilder, errBuilder := metadata.NewTelemetryBuilder(telSettings)
	if errBuilder != nil {
		return nil, errBuilder
	}

	tracer := metadata.Tracer(telSettings)
	spanName := scraperKey + spanNameSep + scraperID.String() + spanNameSep + "ScrapeMetrics"
	otelAttrs := metric.WithAttributeSet(attribute.NewSet(
		attribute.String(receiverKey, receiverID.String()),
		attribute.String(scraperKey, scraperID.String()),
	))

	return func(ctx context.Context) (pmetric.Metrics, error) {
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()

		md, err := delegate(ctx)
		numScrapedMetrics := 0
		numErroredMetrics := 0
		if err != nil {
			telSettings.Logger.Error("Error scraping metrics", zap.Error(err))
			var partialErr scrapererror.PartialScrapeError
			if errors.As(err, &partialErr) {
				numErroredMetrics = partialErr.Failed
				numScrapedMetrics = md.MetricCount()
			}
		} else {
			numScrapedMetrics = md.MetricCount()
		}

		telemetryBuilder.ScraperScrapedMetricPoints.Add(ctx, int64(numScrapedMetrics), otelAttrs)
		telemetryBuilder.ScraperErroredMetricPoints.Add(ctx, int64(numErroredMetrics), otelAttrs)

		// end span according to errors
		if span.IsRecording() {
			span.SetAttributes(
				attribute.String(formatKey, pipeline.SignalMetrics.String()),
				attribute.Int64(scrapedMetricPointsKey, int64(numScrapedMetrics)),
				attribute.Int64(erroredMetricPointsKey, int64(numErroredMetrics)),
			)

			if err != nil {
				span.SetStatus(codes.Error, err.Error())
			}
		}

		return md, err
	}, nil
}

// TODO: This will be implemented in another PR and will be removed soon.
// nolint:unparam
func newObsLogs(delegate scraper.ScrapeLogsFunc, _ component.ID, _ component.ID, _ component.TelemetrySettings) (scraper.ScrapeLogsFunc, error) {
	return func(ctx context.Context) (plog.Logs, error) { return delegate(ctx) }, nil
}
