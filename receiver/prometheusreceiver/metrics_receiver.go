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

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/receiver"
	"github.com/orijtech/promreceiver"
	"github.com/prometheus/prometheus/config"
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
	recv      *promreceiver.Receiver
	cfg       *Configuration
}

var _ receiver.MetricsReceiver = (*Preceiver)(nil)

var (
	errAlreadyStarted         = errors.New("already started the Prometheus receiver")
	errNilMetricsReceiverSink = errors.New("expecting a non-nil MetricsReceiverSink")
	errNilScrapeConfig        = errors.New("expecting a non-nil ScrapeConfig")
)

const (
	prometheusConfigKey = "config"
)

// New creates a new prometheus.Receiver reference.
func New(v *viper.Viper) (*Preceiver, error) {
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
	pr := &Preceiver{cfg: &cfg}
	return pr, nil
}

const metricsSource string = "Prometheus"

// MetricsSource returns the name of the metrics data source.
func (pr *Preceiver) MetricsSource() string {
	return metricsSource
}

// StartMetricsReception is the method that starts Prometheus scraping and it
// is controlled by having previously defined a Configuration using perhaps New.
func (pr *Preceiver) StartMetricsReception(ctx context.Context, nextConsumer consumer.MetricsConsumer) error {
	var err = errAlreadyStarted
	pr.startOnce.Do(func() {
		if nextConsumer == nil {
			err = errNilMetricsReceiverSink
			return
		}

		tms := &promMetricsReceiverToOpenCensusMetricsReceiver{nextConsumer: nextConsumer}
		cfg := pr.cfg
		pr.recv, err = promreceiver.ReceiverFromConfig(
			context.Background(),
			tms,
			cfg.ScrapeConfig,
			promreceiver.WithBufferPeriod(cfg.BufferPeriod),
			promreceiver.WithBufferCount(cfg.BufferCount))
	})
	return err
}

// Flush triggers the Flush method on the underlying Prometheus scrapers and instructs
// them to immediately sned over the metrics they've collected, to the MetricsConsumer.
func (pr *Preceiver) Flush() {
	pr.recv.Flush()
}

// StopMetricsReception stops and cancels the underlying Prometheus scrapers.
func (pr *Preceiver) StopMetricsReception(ctx context.Context) error {
	pr.Flush()
	return pr.recv.Cancel()
}

type promMetricsReceiverToOpenCensusMetricsReceiver struct {
	nextConsumer consumer.MetricsConsumer
}

var _ promreceiver.MetricsSink = (*promMetricsReceiverToOpenCensusMetricsReceiver)(nil)

var errNilRequest = errors.New("expecting a non-nil request")

// ReceiveMetrics is a converter that enables MetricsReceivers to act as MetricsSink.
func (pmrtomr *promMetricsReceiverToOpenCensusMetricsReceiver) ReceiveMetrics(ctx context.Context, ereq *agentmetricspb.ExportMetricsServiceRequest) error {
	if ereq == nil {
		return errNilRequest
	}

	err := pmrtomr.nextConsumer.ConsumeMetricsData(ctx, data.MetricsData{
		Node:     ereq.Node,
		Resource: ereq.Resource,
		Metrics:  ereq.Metrics,
	})
	return err
}
