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

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

// ScraperContext adds the keys used when recording observability metrics to
// the given context returning the newly created context. This context should
// be used in related calls to the obsreport functions so metrics are properly
// recorded.
func ScraperContext(
	ctx context.Context,
	receiverID config.ComponentID,
	scraper config.ComponentID,
) context.Context {
	ctx, _ = tag.New(
		ctx,
		tag.Upsert(obsmetrics.TagKeyReceiver, receiverID.String(), tag.WithTTL(tag.TTLNoPropagation)),
		tag.Upsert(obsmetrics.TagKeyScraper, scraper.String(), tag.WithTTL(tag.TTLNoPropagation)))

	return ctx
}

// StartMetricsScrapeOp is called when a scrape operation is started. The
// returned context should be used in other calls to the obsreport functions
// dealing with the same scrape operation.
func StartMetricsScrapeOp(
	scraperCtx context.Context,
	receiverID config.ComponentID,
	scraper config.ComponentID,
) context.Context {
	spanName := obsmetrics.ScraperPrefix + receiverID.String() + obsmetrics.NameSep + scraper.String() + obsmetrics.ScraperMetricsOperationSuffix
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

	if obsreportconfig.Level != configtelemetry.LevelNone {
		stats.Record(
			scraperCtx,
			obsmetrics.ScraperScrapedMetricPoints.M(int64(numScrapedMetrics)),
			obsmetrics.ScraperErroredMetricPoints.M(int64(numErroredMetrics)))
	}

	// end span according to errors
	if span.IsRecordingEvents() {
		span.AddAttributes(
			trace.StringAttribute(obsmetrics.FormatKey, string(config.MetricsDataType)),
			trace.Int64Attribute(obsmetrics.ScrapedMetricPointsKey, int64(numScrapedMetrics)),
			trace.Int64Attribute(obsmetrics.ErroredMetricPointsKey, int64(numErroredMetrics)),
		)

		span.SetStatus(errToStatus(err))
	}

	span.End()
}
