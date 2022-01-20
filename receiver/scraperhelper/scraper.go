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

package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenthelper"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/model/pdata"
)

var errNilFunc = errors.New("nil scrape func")

// ScrapeFunc scrapes metrics.
type ScrapeFunc func(context.Context) (pdata.Metrics, error)

func (sf ScrapeFunc) Scrape(ctx context.Context) (pdata.Metrics, error) {
	return sf(ctx)
}

// Scraper is the base interface for scrapers.
type Scraper interface {
	component.Component

	// ID returns the scraper id.
	ID() config.ComponentID
	Scrape(context.Context) (pdata.Metrics, error)
}

// ScraperOption apply changes to internal options.
type ScraperOption func(*baseScraper)

// WithStart sets the function that will be called on startup.
func WithStart(start componenthelper.StartFunc) ScraperOption {
	return func(o *baseScraper) {
		o.StartFunc = start
	}
}

// WithShutdown sets the function that will be called on shutdown.
func WithShutdown(shutdown componenthelper.ShutdownFunc) ScraperOption {
	return func(o *baseScraper) {
		o.ShutdownFunc = shutdown
	}
}

var _ Scraper = (*baseScraper)(nil)

type baseScraper struct {
	componenthelper.StartFunc
	componenthelper.ShutdownFunc
	ScrapeFunc
	id config.ComponentID
}

func (b *baseScraper) ID() config.ComponentID {
	return b.id
}

// NewScraper creates a Scraper that calls Scrape at the specified collection interval,
// reports observability information, and passes the scraped metrics to the next consumer.
func NewScraper(name string, scrape ScrapeFunc, options ...ScraperOption) (Scraper, error) {
	if scrape == nil {
		return nil, errNilFunc
	}
	bs := &baseScraper{
		ScrapeFunc: scrape,
		id:         config.NewComponentID(config.Type(name)),
	}
	for _, op := range options {
		op(bs)
	}

	return bs, nil
}
