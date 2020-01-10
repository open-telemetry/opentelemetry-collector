// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusreceiver

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/prometheus/discovery"
	sd_config "github.com/prometheus/prometheus/discovery/config"
	"github.com/prometheus/prometheus/scrape"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
	"github.com/open-telemetry/opentelemetry-collector/receiver/prometheusreceiver/internal"
)

// Preceiver is the type that provides Prometheus scraper/receiver functionality.
type Preceiver struct {
	startOnce sync.Once
	stopOnce  sync.Once
	cfg       *Config
	consumer  consumer.MetricsConsumer
	cancel    context.CancelFunc
	logger    *zap.Logger
}

var _ receiver.MetricsReceiver = (*Preceiver)(nil)

// New creates a new prometheus.Receiver reference.
func newPrometheusReceiver(logger *zap.Logger, cfg *Config, next consumer.MetricsConsumer) *Preceiver {
	pr := &Preceiver{
		cfg:      cfg,
		consumer: next,
		logger:   logger,
	}
	return pr
}

const metricsSource string = "Prometheus"

// MetricsSource returns the name of the metrics data source.
func (pr *Preceiver) MetricsSource() string {
	return metricsSource
}

// Start is the method that starts Prometheus scraping and it
// is controlled by having previously defined a Configuration using perhaps New.
func (pr *Preceiver) Start(host component.Host) error {
	pr.startOnce.Do(func() {
		ctx := host.Context()
		c, cancel := context.WithCancel(ctx)
		pr.cancel = cancel
		// TODO: Use the name from the ReceiverSettings
		c = observability.ContextWithReceiverName(c, pr.cfg.Name())
		var jobsMap *internal.JobsMap
		if !pr.cfg.UseStartTimeMetric {
			jobsMap = internal.NewJobsMap(time.Duration(2 * time.Minute))
		}
		app := internal.NewOcaStore(c, pr.consumer, pr.logger, jobsMap, pr.cfg.UseStartTimeMetric, pr.cfg.Name())
		// need to use a logger with the gokitLog interface
		l := internal.NewZapToGokitLogAdapter(pr.logger)
		scrapeManager := scrape.NewManager(l, app)
		app.SetScrapeManager(scrapeManager)
		discoveryManagerScrape := discovery.NewManager(ctx, l)
		go func() {
			if err := discoveryManagerScrape.Run(); err != nil {
				host.ReportFatalError(err)
			}
		}()
		if err := scrapeManager.ApplyConfig(pr.cfg.PrometheusConfig); err != nil {
			host.ReportFatalError(err)
			return
		}

		// Run the scrape manager.
		syncConfig := make(chan bool)
		errsChan := make(chan error, 1)
		go func() {
			defer close(errsChan)
			<-time.After(100 * time.Millisecond)
			close(syncConfig)
			if err := scrapeManager.Run(discoveryManagerScrape.SyncCh()); err != nil {
				errsChan <- err
			}
		}()
		<-syncConfig
		// By this point we've given time to the scrape manager
		// to start applying its original configuration.

		discoveryCfg := make(map[string]sd_config.ServiceDiscoveryConfig)
		for _, scrapeConfig := range pr.cfg.PrometheusConfig.ScrapeConfigs {
			discoveryCfg[scrapeConfig.JobName] = scrapeConfig.ServiceDiscoveryConfig
		}

		// Now trigger the discovery notification to the scrape manager.
		if err := discoveryManagerScrape.ApplyConfig(discoveryCfg); err != nil {
			errsChan <- err
		}
	})
	return nil
}

// Flush triggers the Flush method on the underlying Prometheus scrapers and instructs
// them to immediately sned over the metrics they've collected, to the MetricsConsumer.
// it's not needed on the new prometheus receiver implementation, let it do nothing
func (pr *Preceiver) Flush() {

}

// Shutdown stops and cancels the underlying Prometheus scrapers.
func (pr *Preceiver) Shutdown() error {
	pr.stopOnce.Do(pr.cancel)
	return nil
}
