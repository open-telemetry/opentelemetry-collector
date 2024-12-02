// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver/internal"
	"go.opentelemetry.io/collector/receiver/scraperhelper/internal/metadata"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
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
)

func newObsMetrics(delegate scraper.ScrapeMetricsFunc, receiverID component.ID, scraperID component.ID, telSettings component.TelemetrySettings) (scraper.ScrapeMetricsFunc, error) {
	telemetryBuilder, errBuilder := metadata.NewTelemetryBuilder(telSettings)
	if errBuilder != nil {
		return nil, errBuilder
	}

	tracer := metadata.Tracer(telSettings)
	spanName := scraperKey + internal.SpanNameSep + scraperID.String() + internal.SpanNameSep + "ScrapeMetrics"
	otelAttrs := metric.WithAttributeSet(attribute.NewSet(
		attribute.String(internal.ReceiverKey, receiverID.String()),
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
				attribute.String(internal.FormatKey, pipeline.SignalMetrics.String()),
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
