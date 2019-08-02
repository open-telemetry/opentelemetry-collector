// Copyright 2019, OpenCensus Authors
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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/scrape"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/receiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/prometheusreceiver/internal"

	"github.com/prometheus/prometheus/config"
	sd_config "github.com/prometheus/prometheus/discovery/config"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// Configuration defines the behavior and targets of the Prometheus scrapers.
type Configuration struct {
	ScrapeConfig *config.Config `mapstructure:"config"`
	BufferPeriod time.Duration  `mapstructure:"buffer_period"`
	BufferCount  int            `mapstructure:"buffer_count"`
}

// Preceiver is the type that provides Prometheus scraper/receiver functionality.
type Preceiver struct {
	startOnce sync.Once
	stopOnce  sync.Once
	cfg       *Configuration
	consumer  consumer.MetricsConsumer
	cancel    context.CancelFunc
	logger    *zap.Logger
}

var _ receiver.MetricsReceiver = (*Preceiver)(nil)

var (
	errNilScrapeConfig = errors.New("expecting a non-nil ScrapeConfig")
)

const (
	prometheusConfigKey = "config"
)

// New creates a new prometheus.Receiver reference.
func New(logger *zap.Logger, v *viper.Viper, next consumer.MetricsConsumer) (*Preceiver, error) {
	var cfg Configuration

	// Unmarshal our config values (using viper's mapstructure)
	err := v.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("prometheus receiver failed to parse config: %s", err)
	}

	// Unmarshal prometheus's config values. Since prometheus uses `yaml` tags, so use `yaml`.
	if !v.IsSet(prometheusConfigKey) {
		return nil, errNilScrapeConfig
	}
	promCfgMap := v.Sub(prometheusConfigKey).AllSettings()
	out, err := yaml.Marshal(promCfgMap)
	if err != nil {
		return nil, fmt.Errorf("prometheus receiver failed to marshal config to yaml: %s", err)
	}
	err = yaml.Unmarshal(out, &cfg.ScrapeConfig)
	if err != nil {
		return nil, fmt.Errorf("prometheus receiver failed to unmarshal yaml to prometheus config: %s", err)
	}
	if len(cfg.ScrapeConfig.ScrapeConfigs) == 0 {
		return nil, errNilScrapeConfig
	}
	pr := &Preceiver{
		cfg:      &cfg,
		consumer: next,
		logger:   logger,
	}
	return pr, nil
}

// New creates a new prometheus.Receiver reference.
func newPrometheusReceiver(logger *zap.Logger, cfg *Configuration, next consumer.MetricsConsumer) *Preceiver {
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

// StartMetricsReception is the method that starts Prometheus scraping and it
// is controlled by having previously defined a Configuration using perhaps New.
func (pr *Preceiver) StartMetricsReception(host receiver.Host) error {
	pr.startOnce.Do(func() {
		ctx := host.Context()
		c, cancel := context.WithCancel(ctx)
		pr.cancel = cancel
		app := internal.NewOcaStore(c, pr.consumer, pr.logger.Sugar())
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
		if err := scrapeManager.ApplyConfig(pr.cfg.ScrapeConfig); err != nil {
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
		for _, scrapeConfig := range pr.cfg.ScrapeConfig.ScrapeConfigs {
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

// StopMetricsReception stops and cancels the underlying Prometheus scrapers.
func (pr *Preceiver) StopMetricsReception() error {
	pr.stopOnce.Do(pr.cancel)
	return nil
}
