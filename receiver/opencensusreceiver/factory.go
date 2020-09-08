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

package opencensusreceiver

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
)

const (
	// The value of "type" key in configuration.
	typeStr = "opencensus"
)

// Factory is the Factory for receiver.
type Factory struct {
}

// Type gets the type of the ocReceiver config created by this Factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for receiver.
func (f *Factory) CreateDefaultConfig() configmodels.Receiver {
	return &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		GRPCServerSettings: configgrpc.GRPCServerSettings{
			NetAddr: confignet.NetAddr{
				Endpoint:  "0.0.0.0:55678",
				Transport: "tcp",
			},
			// We almost write 0 bytes, so no need to tune WriteBufferSize.
			ReadBufferSize: 512 * 1024,
		},
	}
}

// CreateTraceReceiver creates a  trace receiver based on provided config.
func (f *Factory) CreateTraceReceiver(
	_ context.Context,
	_ *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumerOld,
) (component.TraceReceiver, error) {
	r, err := f.createReceiver(cfg)
	if err != nil {
		return nil, err
	}

	r.traceConsumer = nextConsumer

	return r, nil
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func (f *Factory) CreateMetricsReceiver(_ context.Context, _ *zap.Logger, cfg configmodels.Receiver, nextConsumer consumer.MetricsConsumerOld) (component.MetricsReceiver, error) {

	r, err := f.createReceiver(cfg)
	if err != nil {
		return nil, err
	}

	r.metricsConsumer = nextConsumer

	return r, nil
}

func (f *Factory) createReceiver(cfg configmodels.Receiver) (*ocReceiver, error) {
	rCfg := cfg.(*Config)

	// There must be one receiver for both metrics and traces. We maintain a map of
	// receivers per config.

	// Check to see if there is already a receiver for this config.
	receiver, ok := receivers[rCfg]
	if !ok {
		// Build the configuration options.
		opts, err := rCfg.buildOptions()
		if err != nil {
			return nil, err
		}

		// We don't have a receiver, so create one.
		receiver, err = newOpenCensusReceiver(
			rCfg.Name(), rCfg.NetAddr.Transport, rCfg.NetAddr.Endpoint, nil, nil, opts...)
		if err != nil {
			return nil, err
		}
		// Remember the receiver in the map
		receivers[rCfg] = receiver
	}
	return receiver, nil
}

// This is the map of already created OpenCensus receivers for particular configurations.
// We maintain this map because the Factory is asked trace and metric receivers separately
// when it gets CreateTraceReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one ocReceiver object per configuration.
var receivers = map[*Config]*ocReceiver{}
