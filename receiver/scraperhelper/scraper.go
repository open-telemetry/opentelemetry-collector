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

package scraperhelper

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenthelper"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
)

// Scrape metrics.
type ScrapeMetrics func(context.Context) (pdata.MetricSlice, error)

// Scrape resource metrics.
type ScrapeResourceMetrics func(context.Context) (pdata.ResourceMetricsSlice, error)

// ScraperOption apply changes to internal options.
type ScraperOption func(*componenthelper.ComponentSettings)

type BaseScraper interface {
	component.Component

	// Name returns the scraper name
	Name() string
}

// MetricsScraper is an interface for scrapers that scrape metrics.
type MetricsScraper interface {
	BaseScraper
	Scrape(context.Context, string) (pdata.MetricSlice, error)
}

// ResourceMetricsScraper is an interface for scrapers that scrape resource metrics.
type ResourceMetricsScraper interface {
	BaseScraper
	Scrape(context.Context, string) (pdata.ResourceMetricsSlice, error)
}

var _ BaseScraper = (*baseScraper)(nil)

type baseScraper struct {
	component.Component
	name string
}

func (b baseScraper) Name() string {
	return b.name
}

// WithStart sets the function that will be called on startup.
func WithStart(start componenthelper.Start) ScraperOption {
	return func(s *componenthelper.ComponentSettings) {
		s.Start = start
	}
}

// WithShutdown sets the function that will be called on shutdown.
func WithShutdown(shutdown componenthelper.Shutdown) ScraperOption {
	return func(s *componenthelper.ComponentSettings) {
		s.Shutdown = shutdown
	}
}

type metricsScraper struct {
	baseScraper
	ScrapeMetrics
}

var _ MetricsScraper = (*metricsScraper)(nil)

// NewMetricsScraper creates a Scraper that calls Scrape at the specified
// collection interval, reports observability information, and passes the
// scraped metrics to the next consumer.
func NewMetricsScraper(
	name string,
	scrape ScrapeMetrics,
	options ...ScraperOption,
) MetricsScraper {
	set := componenthelper.DefaultComponentSettings()
	for _, op := range options {
		op(set)
	}

	ms := &metricsScraper{
		baseScraper: baseScraper{
			Component: componenthelper.NewComponent(set),
			name:      name,
		},
		ScrapeMetrics: scrape,
	}

	return ms
}

func (ms metricsScraper) Scrape(ctx context.Context, receiverName string) (pdata.MetricSlice, error) {
	ctx = obsreport.ScraperContext(ctx, receiverName, ms.Name())
	ctx = obsreport.StartMetricsScrapeOp(ctx, receiverName, ms.Name())
	metrics, err := ms.ScrapeMetrics(ctx)
	obsreport.EndMetricsScrapeOp(ctx, metrics.Len(), err)
	return metrics, err
}

type resourceMetricsScraper struct {
	baseScraper
	ScrapeResourceMetrics
}

var _ ResourceMetricsScraper = (*resourceMetricsScraper)(nil)

// NewResourceMetricsScraper creates a Scraper that calls Scrape at the
// specified collection interval, reports observability information, and
// passes the scraped resource metrics to the next consumer.
func NewResourceMetricsScraper(
	name string,
	scrape ScrapeResourceMetrics,
	options ...ScraperOption,
) ResourceMetricsScraper {
	set := componenthelper.DefaultComponentSettings()
	for _, op := range options {
		op(set)
	}

	rms := &resourceMetricsScraper{
		baseScraper: baseScraper{
			Component: componenthelper.NewComponent(set),
			name:      name,
		},
		ScrapeResourceMetrics: scrape,
	}

	return rms
}

func (rms resourceMetricsScraper) Scrape(ctx context.Context, receiverName string) (pdata.ResourceMetricsSlice, error) {
	ctx = obsreport.ScraperContext(ctx, receiverName, rms.Name())
	ctx = obsreport.StartMetricsScrapeOp(ctx, receiverName, rms.Name())
	resourceMetrics, err := rms.ScrapeResourceMetrics(ctx)
	obsreport.EndMetricsScrapeOp(ctx, metricCount(resourceMetrics), err)
	return resourceMetrics, err
}

func metricCount(resourceMetrics pdata.ResourceMetricsSlice) int {
	count := 0

	for i := 0; i < resourceMetrics.Len(); i++ {
		ilm := resourceMetrics.At(i).InstrumentationLibraryMetrics()
		for j := 0; j < ilm.Len(); j++ {
			count += ilm.At(j).Metrics().Len()
		}
	}

	return count
}
