// Copyright The OpenTelemetry Authors
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
	"runtime"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
)

// This file implements Factory for VMMetrics receiver.

const (
	// The value of "type" key in configuration.
	typeStr = "vmmetrics"
)

// Factory is the Factory for receiver.
type Factory struct {
}

// Type gets the type of the Receiver config created by this Factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CustomUnmarshaler returns custom unmarshaler for this config.
func (f *Factory) CustomUnmarshaler() component.CustomUnmarshaler {
	return nil
}

// CreateDefaultConfig creates the default configuration for receiver.
func (f *Factory) CreateDefaultConfig() configmodels.Receiver {
	return &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

// CreateTraceReceiver creates a trace receiver based on provided config.
func (f *Factory) CreateTraceReceiver(
	ctx context.Context,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumerOld,
) (component.TraceReceiver, error) {
	// VMMetrics does not support traces
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func (f *Factory) CreateMetricsReceiver(
	logger *zap.Logger,
	config configmodels.Receiver,
	consumer consumer.MetricsConsumerOld,
) (component.MetricsReceiver, error) {

	if runtime.GOOS != "linux" {
		// TODO: add support for other platforms.
		return nil, errors.New("vmmetrics receiver is only supported on linux")
	}
	cfg := config.(*Config)

	vmc, err := NewVMMetricsCollector(cfg.ScrapeInterval, cfg.MountPoint, cfg.ProcessMountPoint, cfg.MetricPrefix, consumer)
	if err != nil {
		return nil, err
	}

	vmr := &Receiver{
		vmc: vmc,
	}
	return vmr, nil
}
