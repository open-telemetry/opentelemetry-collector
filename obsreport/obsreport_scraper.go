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

package obsreport

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/receiver/scrapererror"
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

const (
	scraperPrefix                 = scraperKey + nameSep
	scraperMetricsOperationSuffix = nameSep + "MetricsScraped"
)

var (
	tagKeyScraper, _ = tag.NewKey(scraperKey)

	mScraperScrapedMetricPoints = stats.Int64(
		scraperPrefix+scrapedMetricPointsKey,
		"Number of metric points successfully scraped.",
		stats.UnitDimensionless)
	mScraperErroredMetricPoints = stats.Int64(
		scraperPrefix+erroredMetricPointsKey,
		"Number of metric points that were unable to be scraped.",
		stats.UnitDimensionless)
)

// ScraperContext adds the keys used when recording observability metrics to
// the given context returning the newly created context. This context should
// be used in related calls to the obsreport functions so metrics are properly
// recorded.
func ScraperContext(
	ctx context.Context,
	receiver string,
	scraper string,
) context.Context {
	ctx, _ = tag.New(ctx, tag.Upsert(tagKeyReceiver, receiver, tag.WithTTL(tag.TTLNoPropagation)))
	if scraper != "" {
		ctx, _ = tag.New(ctx, tag.Upsert(tagKeyScraper, scraper, tag.WithTTL(tag.TTLNoPropagation)))
	}

	return ctx
}

// StartMetricsScrapeOp is called when a scrape operation is started. The
// returned context should be used in other calls to the obsreport functions
// dealing with the same scrape operation.
func StartMetricsScrapeOp(
	scraperCtx context.Context,
	receiver string,
	scraper string,
) context.Context {
	scraperName := receiver
	if scraper != "" {
		scraperName += "/" + scraper
	}

	spanName := scraperPrefix + scraperName + scraperMetricsOperationSuffix
	ctx, _ := trace.StartSpan(scraperCtx, spanName)
	return ctx
}

// EndMetricsScrapeOp completes the scrape operation that was started with
// StartMetricsScrapeOp.
func EndMetricsScrapeOp(
	scraperCtx context.Context,
	numScrapedMetrics int,
	err error,
) {
	numErroredMetrics := 0
	if err != nil {
		if partialErr, isPartial := err.(scrapererror.PartialScrapeError); isPartial {
			numErroredMetrics = partialErr.Failed
		} else {
			numErroredMetrics = numScrapedMetrics
			numScrapedMetrics = 0
		}
	}

	span := trace.FromContext(scraperCtx)

	if gLevel != configtelemetry.LevelNone {
		stats.Record(
			scraperCtx,
			mScraperScrapedMetricPoints.M(int64(numScrapedMetrics)),
			mScraperErroredMetricPoints.M(int64(numErroredMetrics)))
	}

	// end span according to errors
	if span.IsRecordingEvents() {
		span.AddAttributes(
			trace.StringAttribute(formatKey, string(configmodels.MetricsDataType)),
			trace.Int64Attribute(scrapedMetricPointsKey, int64(numScrapedMetrics)),
			trace.Int64Attribute(erroredMetricPointsKey, int64(numErroredMetrics)),
		)

		span.SetStatus(errToStatus(err))
	}

	span.End()
}
