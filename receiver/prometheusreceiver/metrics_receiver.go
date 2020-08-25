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

package prometheusreceiver

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/prometheus/discovery"
	sdconfig "github.com/prometheus/prometheus/discovery/config"
	"github.com/prometheus/prometheus/scrape"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/receiver/prometheusreceiver/internal"
)

// pReceiver is the type that provides Prometheus scraper/receiver functionality.
type pReceiver struct {
	startOnce sync.Once
	stopOnce  sync.Once
	cfg       *Config
	consumer  consumer.MetricsConsumer
	cancel    context.CancelFunc
	logger    *zap.Logger
}

// New creates a new prometheus.Receiver reference.
func newPrometheusReceiver(logger *zap.Logger, cfg *Config, next consumer.MetricsConsumer) *pReceiver {
	pr := &pReceiver{
		cfg:      cfg,
		consumer: next,
		logger:   logger,
	}
	return pr
}

// Start is the method that starts Prometheus scraping and it
// is controlled by having previously defined a Configuration using perhaps New.
func (pr *pReceiver) Start(_ context.Context, host component.Host) error {
	pr.startOnce.Do(func() {
		ctx := context.Background()
		c, cancel := context.WithCancel(ctx)
		pr.cancel = cancel
		c = obsreport.ReceiverContext(c, pr.cfg.Name(), "http", pr.cfg.Name())
		var jobsMap *internal.JobsMap
		if !pr.cfg.UseStartTimeMetric {
			jobsMap = internal.NewJobsMap(2 * time.Minute)
		}
		app := internal.NewOcaStore(c, pr.consumer, pr.logger, jobsMap, pr.cfg.UseStartTimeMetric, pr.cfg.StartTimeMetricRegex, pr.cfg.Name())
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

		discoveryCfg := make(map[string]sdconfig.ServiceDiscoveryConfig)
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

// Shutdown stops and cancels the underlying Prometheus scrapers.
func (pr *pReceiver) Shutdown(context.Context) error {
	pr.stopOnce.Do(pr.cancel)
	return nil
}
