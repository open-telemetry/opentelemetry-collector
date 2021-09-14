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
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/receiver/scrapererror"
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

// Scraper is the base interface for scrapers.
type Scraper interface {
	component.Component

	// ID returns the scraper id.
	ID() config.ComponentID
	Scrape(context.Context, config.ComponentID) (pdata.Metrics, error)
}

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

var _ Scraper = (*metricsScraper)(nil)

// NewMetricsScraper creates a Scraper that calls Scrape at the specified
// collection interval, reports observability information, and passes the
// scraped metrics to the next consumer.
func NewMetricsScraper(
	name string,
	scrape ScrapeMetrics,
	options ...ScraperOption,
) Scraper {
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

func (ms metricsScraper) Scrape(ctx context.Context, receiverID config.ComponentID) (pdata.Metrics, error) {
	scrp := obsreport.NewScraper(obsreport.ScraperSettings{ReceiverID: receiverID, Scraper: ms.ID()})
	ctx = scrp.StartMetricsOp(ctx)
	metrics, err := ms.ScrapeMetrics(ctx)
	count := 0
	md := pdata.Metrics{}
	if err == nil || scrapererror.IsPartialScrapeError(err) {
		md = pdata.NewMetrics()
		metrics.MoveAndAppendTo(md.ResourceMetrics().AppendEmpty().InstrumentationLibraryMetrics().AppendEmpty().Metrics())
		count = md.MetricCount()
	}
	scrp.EndMetricsOp(ctx, count, err)
	return md, err
}

type resourceMetricsScraper struct {
	baseScraper
	ScrapeResourceMetrics
}

var _ Scraper = (*resourceMetricsScraper)(nil)

// NewResourceMetricsScraper creates a Scraper that calls Scrape at the
// specified collection interval, reports observability information, and
// passes the scraped resource metrics to the next consumer.
func NewResourceMetricsScraper(
	id config.ComponentID,
	scrape ScrapeResourceMetrics,
	options ...ScraperOption,
) Scraper {
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

func (rms resourceMetricsScraper) Scrape(ctx context.Context, receiverID config.ComponentID) (pdata.Metrics, error) {
	scrp := obsreport.NewScraper(obsreport.ScraperSettings{ReceiverID: receiverID, Scraper: rms.ID()})
	ctx = scrp.StartMetricsOp(ctx)
	resourceMetrics, err := rms.ScrapeResourceMetrics(ctx)

	count := 0
	md := pdata.Metrics{}
	if err == nil || scrapererror.IsPartialScrapeError(err) {
		md = pdata.NewMetrics()
		resourceMetrics.MoveAndAppendTo(md.ResourceMetrics())
		count = md.MetricCount()
	}

	scrp.EndMetricsOp(ctx, count, err)
	return md, err
}
