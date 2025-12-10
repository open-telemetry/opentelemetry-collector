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
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
	"go.opentelemetry.io/collector/scraper/scraperhelper/internal/metadata"
)

const (
	// scrapedLogRecordsKey used to identify log records scraped by the
	// Collector.
	scrapedLogRecordsKey = "scraped_log_records"
	// erroredLogRecordsKey used to identify log records errored (i.e.
	// unable to be scraped) by the Collector.
	erroredLogRecordsKey = "errored_log_records"
)

func wrapObsLogs(sc scraper.Logs, receiverID, scraperID component.ID, set component.TelemetrySettings) (scraper.Logs, error) {
	telemetryBuilder, errBuilder := metadata.NewTelemetryBuilder(set)
	if errBuilder != nil {
		return nil, errBuilder
	}

	tracer := metadata.Tracer(set)
	spanName := scraperKey + spanNameSep + scraperID.String() + spanNameSep + "ScrapeLogs"
	otelAttrs := metric.WithAttributeSet(attribute.NewSet(
		attribute.String(receiverKey, receiverID.String()),
		attribute.String(scraperKey, scraperID.String()),
	))

	scraperFuncs := func(ctx context.Context) (plog.Logs, error) {
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()

		md, err := sc.ScrapeLogs(ctx)
		numScrapedLogs := 0
		numErroredLogs := 0
		if err != nil {
			set.Logger.Error("Error scraping logs", zap.Error(err))
			var partialErr scrapererror.PartialScrapeError
			if errors.As(err, &partialErr) {
				numErroredLogs = partialErr.Failed
				numScrapedLogs = md.LogRecordCount()
			}
		} else {
			numScrapedLogs = md.LogRecordCount()
		}

		telemetryBuilder.ScraperScrapedLogRecords.Add(ctx, int64(numScrapedLogs), otelAttrs)
		telemetryBuilder.ScraperErroredLogRecords.Add(ctx, int64(numErroredLogs), otelAttrs)

		// end span according to errors
		if span.IsRecording() {
			span.SetAttributes(
				attribute.String(formatKey, pipeline.SignalMetrics.String()),
				attribute.Int64(scrapedLogRecordsKey, int64(numScrapedLogs)),
				attribute.Int64(erroredLogRecordsKey, int64(numErroredLogs)),
			)

			if err != nil {
				span.SetStatus(codes.Error, err.Error())
			}
		}

		return md, err
	}

	return scraper.NewLogs(scraperFuncs, scraper.WithStart(sc.Start), scraper.WithShutdown(sc.Shutdown))
}
