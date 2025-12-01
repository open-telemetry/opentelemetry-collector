// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package xscraperhelper provides utilities for scrapers.
package xscraperhelper // import "go.opentelemetry.io/collector/scraper/scraperhelper/xscraperhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/scraper/scrapererror"
	"go.opentelemetry.io/collector/scraper/scraperhelper/internal/metadata"
	"go.opentelemetry.io/collector/scraper/xscraper"
)

const (
	// scraperKey used to identify scrapers in metrics and traces.
	scraperKey  = "scraper"
	spanNameSep = "/"
	// receiverKey used to identify receivers in metrics and traces.
	receiverKey = "receiver"
	// FormatKey used to identify the format of the data received.
	formatKey = "format"

	// scrapedProfileRecordsKey used to identify profile records scraped by the
	// Collector.
	scrapedProfileRecordsKey = "scraped_profile_records"
	// erroredProfileRecordsKey used to identify profile records errored (i.e.
	// unable to be scraped) by the Collector.
	erroredProfileRecordsKey = "errored_profile_records"
)

func wrapObsProfiles(sc xscraper.Profiles, receiverID, scraperID component.ID, set component.TelemetrySettings) (xscraper.Profiles, error) {
	telemetryBuilder, errBuilder := metadata.NewTelemetryBuilder(set)
	if errBuilder != nil {
		return nil, errBuilder
	}

	tracer := metadata.Tracer(set)
	spanName := scraperKey + spanNameSep + scraperID.String() + spanNameSep + "ScrapeProfiles"
	otelAttrs := metric.WithAttributeSet(attribute.NewSet(
		attribute.String(receiverKey, receiverID.String()),
		attribute.String(scraperKey, scraperID.String()),
	))

	scraperFuncs := func(ctx context.Context) (pprofile.Profiles, error) {
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()

		md, err := sc.ScrapeProfiles(ctx)
		numScrapedProfiles := 0
		numErroredProfiles := 0
		if err != nil {
			set.Logger.Error("Error scraping profiles", zap.Error(err))
			var partialErr scrapererror.PartialScrapeError
			if errors.As(err, &partialErr) {
				numErroredProfiles = partialErr.Failed
				numScrapedProfiles = md.SampleCount()
			}
		} else {
			numScrapedProfiles = md.SampleCount()
		}

		telemetryBuilder.ScraperScrapedProfileRecords.Add(ctx, int64(numScrapedProfiles), otelAttrs)
		telemetryBuilder.ScraperErroredProfileRecords.Add(ctx, int64(numErroredProfiles), otelAttrs)

		// end span according to errors
		if span.IsRecording() {
			span.SetAttributes(
				attribute.String(formatKey, pipeline.SignalMetrics.String()),
				attribute.Int64(scrapedProfileRecordsKey, int64(numScrapedProfiles)),
				attribute.Int64(erroredProfileRecordsKey, int64(numErroredProfiles)),
			)

			if err != nil {
				span.SetStatus(codes.Error, err.Error())
			}
		}

		return md, err
	}

	return xscraper.NewProfiles(scraperFuncs, xscraper.WithStart(sc.Start), xscraper.WithShutdown(sc.Shutdown))
}
