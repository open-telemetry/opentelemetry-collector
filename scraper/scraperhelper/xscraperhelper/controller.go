// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package xscraperhelper provides utilities for scrapers.
package xscraperhelper // import "go.opentelemetry.io/collector/scraper/scraperhelper/xscraperhelper"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/xreceiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scrapererror"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
	"go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"
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
)

type factoryWithConfig struct {
	f   xscraper.Factory
	cfg component.Config
}

type controllerOptions struct {
	tickerCh            <-chan time.Time
	factoriesWithConfig []factoryWithConfig
}

// ControllerOption apply changes to internal options.
type ControllerOption interface {
	apply(*controllerOptions)
}

type optionFunc func(*controllerOptions)

func (of optionFunc) apply(e *controllerOptions) {
	of(e)
}

// AddProfilesScraper configures the xscraper.Profiles to be called with the specified options,
// and at the specified collection interval.
//
// Observability information will be reported, and the scraped profiles
// will be passed to the next consumer.
func AddProfilesScraper(t component.Type, sc xscraper.Profiles) ControllerOption {
	f := xscraper.NewFactory(t, nil,
		xscraper.WithProfiles(func(context.Context, scraper.Settings, component.Config) (xscraper.Profiles, error) {
			return sc, nil
		}, component.StabilityLevelDevelopment))
	return AddFactoryWithConfig(f, nil)
}

// AddFactoryWithConfig configures the scraper.Factory and associated config that
// will be used to create a new scraper. The created scraper will be called with
// the specified options, and at the specified collection interval.
//
// Observability information will be reported, and the scraped metrics
// will be passed to the next consumer.
func AddFactoryWithConfig(f xscraper.Factory, cfg component.Config) ControllerOption {
	return optionFunc(func(o *controllerOptions) {
		o.factoriesWithConfig = append(o.factoriesWithConfig, factoryWithConfig{f: f, cfg: cfg})
	})
}

// WithTickerChannel allows you to override the scraper controller's ticker
// channel to specify when scrape is called. This is only expected to be
// used by tests.
func WithTickerChannel(tickerCh <-chan time.Time) ControllerOption {
	return optionFunc(func(o *controllerOptions) {
		o.tickerCh = tickerCh
	})
}

// NewProfilesController creates a receiver.Profiles with the configured options, that can control multiple xscraper.Profiles.
func NewProfilesController(cfg *scraperhelper.ControllerConfig,
	rSet receiver.Settings,
	nextConsumer xconsumer.Profiles,
	options ...ControllerOption,
) (xreceiver.Profiles, error) {
	co := getOptions(options)
	scrapers := make([]xscraper.Profiles, 0, len(co.factoriesWithConfig))
	for _, fwc := range co.factoriesWithConfig {
		set := controller.GetSettings(fwc.f.Type(), rSet)
		s, err := fwc.f.CreateProfiles(context.Background(), set, fwc.cfg)
		if err != nil {
			return nil, err
		}
		s, err = wrapObsProfiles(s, rSet.ID, set.ID, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		scrapers = append(scrapers, s)
	}
	return controller.NewController[xscraper.Profiles](
		cfg, rSet, scrapers, func(c *controller.Controller[xscraper.Profiles]) { scrapeProfiles(c, nextConsumer) }, co.tickerCh)
}

func getOptions(options []ControllerOption) controllerOptions {
	co := controllerOptions{}
	for _, op := range options {
		op.apply(&co)
	}
	return co
}

func scrapeProfiles(c *controller.Controller[xscraper.Profiles], nextConsumer xconsumer.Profiles) {
	ctx, done := controller.WithScrapeContext(c.Timeout)
	defer done()

	profiles := pprofile.NewProfiles()
	for i := range c.Scrapers {
		md, err := c.Scrapers[i].ScrapeProfiles(ctx)
		if err != nil && !scrapererror.IsPartialScrapeError(err) {
			continue
		}
		md.ResourceProfiles().MoveAndAppendTo(profiles.ResourceProfiles())
	}

	// TODO: Add proper receiver observability for profiles when receiverhelper supports it
	// For now, we skip the obs report and just consume the profiles directly
	_ = nextConsumer.ConsumeProfiles(ctx, profiles)
}
