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
	"time"

	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/scrape"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/prometheusreceiver/internal"
)

// pReceiver is the type that provides Prometheus scraper/receiver functionality.
type pReceiver struct {
	cfg        *Config
	consumer   consumer.Metrics
	cancelFunc context.CancelFunc

	logger        *zap.Logger
	scrapeManager *scrape.Manager
	ocaStore      *internal.OcaStore
}

// New creates a new prometheus.Receiver reference.
func newPrometheusReceiver(logger *zap.Logger, cfg *Config, next consumer.Metrics) *pReceiver {
	pr := &pReceiver{
		cfg:      cfg,
		consumer: next,
		logger:   logger,
	}
	return pr
}

// Start is the method that starts Prometheus scraping and it
// is controlled by having previously defined a Configuration using perhaps New.
func (r *pReceiver) Start(_ context.Context, host component.Host) error {
	discoveryCtx, cancel := context.WithCancel(context.Background())
	r.cancelFunc = cancel

	logger := internal.NewZapToGokitLogAdapter(r.logger)

	discoveryManager := discovery.NewManager(discoveryCtx, logger)
	discoveryCfg := make(map[string]discovery.Configs)
	for _, scrapeConfig := range r.cfg.PrometheusConfig.ScrapeConfigs {
		discoveryCfg[scrapeConfig.JobName] = scrapeConfig.ServiceDiscoveryConfigs
	}
	if err := discoveryManager.ApplyConfig(discoveryCfg); err != nil {
		return err
	}
	go func() {
		if err := discoveryManager.Run(); err != nil {
			r.logger.Error("Discovery manager failed", zap.Error(err))
			host.ReportFatalError(err)
		}
	}()

	var jobsMap *internal.JobsMap
	if !r.cfg.UseStartTimeMetric {
		jobsMap = internal.NewJobsMap(2 * time.Minute)
	}
	// Per component.Component Start instructions, for async operations we should not use the
	// incoming context, it may get cancelled.
	r.ocaStore = internal.NewOcaStore(
		context.Background(),
		r.consumer,
		r.logger,
		jobsMap,
		r.cfg.UseStartTimeMetric,
		r.cfg.StartTimeMetricRegex,
		r.cfg.ID(),
		r.cfg.PrometheusConfig.GlobalConfig.ExternalLabels,
	)
	r.scrapeManager = scrape.NewManager(logger, r.ocaStore)
	r.ocaStore.SetScrapeManager(r.scrapeManager)
	if err := r.scrapeManager.ApplyConfig(r.cfg.PrometheusConfig); err != nil {
		return err
	}
	go func() {
		if err := r.scrapeManager.Run(discoveryManager.SyncCh()); err != nil {
			r.logger.Error("Scrape manager failed", zap.Error(err))
			host.ReportFatalError(err)
		}
	}()
	return nil
}

// Shutdown stops and cancels the underlying Prometheus scrapers.
func (r *pReceiver) Shutdown(context.Context) error {
	r.cancelFunc()
	// ocaStore (and internally metadataService) needs to stop first to prevent deadlocks.
	// When stopping scrapeManager it waits for all scrapes to terminate. However during
	// scraping metadataService calls scrapeManager.AllTargets() which acquires
	// the same lock that's acquired when scrapeManager is stopped.
	r.ocaStore.Close()
	r.scrapeManager.Stop()
	return nil
}
