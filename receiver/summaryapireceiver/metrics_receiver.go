package summaryapireceiver

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/open-telemetry/opentelemetry-service/consumer"
)

var (
	errAlreadyStarted     = errors.New("already started")
	errAlreadyStopped     = errors.New("already stopped")
	errNilMetricsConsumer = errors.New("expecting a non-nil MetricsConsumer")
)

// Configuration defines the behavior and targets of the Summary API scrapers.
type Configuration struct {
	scrapeInterval  time.Duration `mapstructure:"scrape_interval"`
	kubeletEndpoint string        `mapstructure:"kubelet_endpoint"`
	metricPrefix    string        `mapstructure:"metric_prefix"`
}

// Receiver is the type used to handle metrics from VM metrics.
type Receiver struct {
	mu sync.Mutex

	sac *SummaryApiCollector

	stopOnce  sync.Once
	startOnce sync.Once
}

// New creates a new summaryapireceiver.Receiver reference.
func New(v *viper.Viper, consumer consumer.MetricsConsumer) (*Receiver, error) {
	if consumer == nil {
		return nil, errNilMetricsConsumer
	}

	var cfg Configuration

	// Unmarshal our config values (using viper's mapstructure)
	err := unmarshal(&cfg, v.AllSettings())
	if err != nil {
		return nil, fmt.Errorf("vmmetrics receiver failed to parse config: %s", err)
	}

	sac, err := NewSummaryApiCollector(cfg.scrapeInterval, cfg.kubeletEndpoint, cfg.metricPrefix, consumer)
	if err != nil {
		return nil, err
	}

	sar := &Receiver{
		sac: sac,
	}
	return sar, nil
}

const metricsSource string = "summaryapi"

// MetricsSource is the summaryapi.
func (sar *Receiver) MetricsSource() string {
	return metricsSource
}

// StartMetricsReception scrapes VM metrics based on the OS platform.
func (sar *Receiver) StartMetricsReception(ctx context.Context, asyncErrorChan chan<- error) error {
	sar.mu.Lock()
	defer sar.mu.Unlock()

	var err = errAlreadyStarted
	sar.startOnce.Do(func() {
		sar.sac.StartCollection()
		err = nil
	})
	return err
}

// StopMetricsReception stops and cancels the underlying VM metrics scrapers.
func (sar *Receiver) StopMetricsReception(ctx context.Context) error {
	sar.mu.Lock()
	defer sar.mu.Unlock()

	var err = errAlreadyStopped
	sar.stopOnce.Do(func() {
		sar.sac.StopCollection()
		err = nil
	})
	return err
}

// TODO(songya): investigate why viper.Unmarshal didn't work, remove this method and use viper.Unmarshal instead.
func unmarshal(cfg *Configuration, settings map[string]interface{}) error {
	if interval, ok := settings["scrape_interval"]; ok {
		intervalInSecs := interval.(int)
		cfg.scrapeInterval = time.Duration(intervalInSecs * int(time.Second))
	}
	if kubeletEndpoint, ok := settings["kubelet_endpoint"]; ok {
		cfg.kubeletEndpoint = kubeletEndpoint.(string)
	}
	if prefix, ok := settings["metric_prefix"]; ok {
		cfg.metricPrefix = prefix.(string)
	}
	return nil
}
