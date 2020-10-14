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
)

const (
	// Key used to identify scrapers in metrics and traces.
	ScraperKey = "scraper"

	// Key used to identify metric points scraped by the Collector.
	ScrapedMetricPointsKey = "scraped_metric_points"
	// Key used to identify metric points errored (ie.: unable to be
	// scraped) by the Collector.
	ErroredMetricPointsKey = "refused_metric_points"
)

var (
	tagKeyScraper, _ = tag.NewKey(ScraperKey)

	scraperPrefix                 = ScraperKey + nameSep
	scraperMetricsOperationSuffix = nameSep + "MetricsScraped"

	mScraperScrapedMetricPoints = stats.Int64(
		scraperPrefix+ScrapedMetricPointsKey,
		"Number of metric points successfully scraped.",
		stats.UnitDimensionless)
	mScraperErroredMetricPoints = stats.Int64(
		scraperPrefix+ErroredMetricPointsKey,
		"Number of metric points that were unable to be scraped.",
		stats.UnitDimensionless)
)

// ScraperContext adds the keys used when recording observability metrics to
// the given context returning the newly created context. This context should
// be used in related calls to the obsreport functions so metrics are properly
// recorded.
func ScraperContext(
	ctx context.Context,
	scraper string,
) context.Context {
	ctx, _ = tag.New(ctx, tag.Upsert(tagKeyScraper, scraper, tag.WithTTL(tag.TTLNoPropagation)))
	return ctx
}

// StartMetricsScrapeOp is called when a scrape operation is started. The
// returned context should be used in other calls to the obsreport functions
// dealing with the same scrape operation.
func StartMetricsScrapeOp(
	scraperCtx context.Context,
	scraper string,
) context.Context {
	spanName := scraperPrefix + scraper + scraperMetricsOperationSuffix
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
	numErroredMetrics := 0 // TODO determine based on if error is scrapeerr

	span := trace.FromContext(scraperCtx)

	if useNew {
		stats.Record(
			scraperCtx,
			mScraperScrapedMetricPoints.M(int64(numScrapedMetrics)),
			mScraperErroredMetricPoints.M(int64(numErroredMetrics)))
	}

	// end span according to errors
	if span.IsRecordingEvents() {
		span.AddAttributes(
			trace.StringAttribute(FormatKey, string(configmodels.MetricsDataType)),
			trace.Int64Attribute(ScrapedMetricPointsKey, int64(numScrapedMetrics)),
			trace.Int64Attribute(ErroredMetricPointsKey, int64(numErroredMetrics)),
		)

		span.SetStatus(errToStatus(err))
	}

	span.End()
}
