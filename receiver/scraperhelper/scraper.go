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
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
)

// ScrapeMetrics scrapes metrics.
type ScrapeMetrics func(context.Context) (pdata.MetricSlice, error)

// ScrapeResourceMetrics scrapes resource metrics.
type ScrapeResourceMetrics func(context.Context) (pdata.ResourceMetricsSlice, error)

type baseSettings struct {
	componentOptions []componenthelper.Option
}

// ScraperOption apply changes to internal options.
type ScraperOption func(*baseSettings)

// BaseScraper is the base interface for scrapers.
type BaseScraper interface {
	component.Component

	// ID returns the scraper id.
	ID() config.ComponentID
}

// MetricsScraper is an interface for scrapers that scrape metrics.
type MetricsScraper interface {
	BaseScraper
	Scrape(context.Context, config.ComponentID) (pdata.MetricSlice, error)
}

// ResourceMetricsScraper is an interface for scrapers that scrape resource metrics.
type ResourceMetricsScraper interface {
	BaseScraper
	Scrape(context.Context, config.ComponentID) (pdata.ResourceMetricsSlice, error)
}

var _ BaseScraper = (*baseScraper)(nil)

type baseScraper struct {
	component.Component
	id config.ComponentID
}

func (b baseScraper) ID() config.ComponentID {
	return b.id
}

// WithStart sets the function that will be called on startup.
func WithStart(start componenthelper.StartFunc) ScraperOption {
	return func(o *baseSettings) {
		o.componentOptions = append(o.componentOptions, componenthelper.WithStart(start))
	}
}

// WithShutdown sets the function that will be called on shutdown.
func WithShutdown(shutdown componenthelper.ShutdownFunc) ScraperOption {
	return func(o *baseSettings) {
		o.componentOptions = append(o.componentOptions, componenthelper.WithShutdown(shutdown))
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
	set := &baseSettings{}
	for _, op := range options {
		op(set)
	}

	ms := &metricsScraper{
		baseScraper: baseScraper{
			Component: componenthelper.New(set.componentOptions...),
			id:        config.NewID(config.Type(name)),
		},
		ScrapeMetrics: scrape,
	}

	return ms
}

func (ms metricsScraper) Scrape(ctx context.Context, receiverID config.ComponentID) (pdata.MetricSlice, error) {
	ctx = obsreport.ScraperContext(ctx, receiverID, ms.ID())
	ctx = obsreport.StartMetricsScrapeOp(ctx, receiverID, ms.ID())
	metrics, err := ms.ScrapeMetrics(ctx)
	count := 0
	if err == nil {
		count = metrics.Len()
	}
	obsreport.EndMetricsScrapeOp(ctx, count, err)
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
	id config.ComponentID,
	scrape ScrapeResourceMetrics,
	options ...ScraperOption,
) ResourceMetricsScraper {
	set := &baseSettings{}
	for _, op := range options {
		op(set)
	}

	rms := &resourceMetricsScraper{
		baseScraper: baseScraper{
			Component: componenthelper.New(set.componentOptions...),
			id:        id,
		},
		ScrapeResourceMetrics: scrape,
	}

	return rms
}

func (rms resourceMetricsScraper) Scrape(ctx context.Context, receiverID config.ComponentID) (pdata.ResourceMetricsSlice, error) {
	ctx = obsreport.ScraperContext(ctx, receiverID, rms.ID())
	ctx = obsreport.StartMetricsScrapeOp(ctx, receiverID, rms.ID())
	resourceMetrics, err := rms.ScrapeResourceMetrics(ctx)
	count := 0
	if err == nil {
		count = metricCount(resourceMetrics)
	}
	obsreport.EndMetricsScrapeOp(ctx, count, err)
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
