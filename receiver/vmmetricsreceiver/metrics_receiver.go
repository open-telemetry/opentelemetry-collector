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

package vmmetricsreceiver

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/census-instrumentation/opencensus-service/consumer"
)

var (
	errAlreadyStarted     = errors.New("already started")
	errAlreadyStopped     = errors.New("already stopped")
	errNilMetricsConsumer = errors.New("expecting a non-nil MetricsConsumer")
)

// Configuration defines the behavior and targets of the VM metrics scrapers.
type Configuration struct {
	scrapeInterval time.Duration `mapstructure:"scrape_interval"`
	mountPoint     string        `mapstructure:"mount_point"`
	metricPrefix   string        `mapstructure:"metric_prefix"`
}

// Receiver is the type used to handle metrics from VM metrics.
type Receiver struct {
	mu sync.Mutex

	vmc *VMMetricsCollector

	stopOnce  sync.Once
	startOnce sync.Once
}

// New creates a new vmmetricsreceiver.Receiver reference.
func New(v *viper.Viper, consumer consumer.MetricsConsumer) (*Receiver, error) {
	if consumer == nil {
		return nil, errNilMetricsConsumer
	}

	var cfg Configuration

	// Unmarshal our config values (using viper's mapstructure)
	err := v.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("vmmetrics receiver failed to parse config: %s", err)
	}

	vmc, err := NewVMMetricsCollector(cfg.scrapeInterval, cfg.mountPoint, cfg.metricPrefix, consumer)
	if err != nil {
		return nil, err
	}

	vmr := &Receiver{
		vmc: vmc,
	}
	return vmr, nil
}

// StartMetricsReception scrapes VM metrics based on the OS platform.
func (vmr *Receiver) StartMetricsReception(ctx context.Context, asyncErrorChan chan<- error) error {
	vmr.mu.Lock()
	defer vmr.mu.Unlock()

	var err = errAlreadyStarted
	vmr.startOnce.Do(func() {
		switch runtime.GOOS {
		case "linux":
			vmr.vmc.StartCollection()
		case "darwin", "freebsd", "windows":
			// TODO: add support for other platforms.
			return
		}

		err = nil
	})
	return err
}

// StopMetricsReception stops and cancels the underlying VM metrics scrapers.
func (vmr *Receiver) StopMetricsReception(ctx context.Context) error {
	vmr.mu.Lock()
	defer vmr.mu.Unlock()

	var err = errAlreadyStopped
	vmr.stopOnce.Do(func() {
		vmr.vmc.StopCollection()
		err = nil
	})
	return err
}
