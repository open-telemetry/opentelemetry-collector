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

package opencensusreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/sharedcomponent"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const typeStr = "opencensus"

// NewFactory creates a new OpenCensus receiver factory.
func NewFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		receiverhelper.WithTraces(createTracesReceiver),
		receiverhelper.WithMetrics(createMetricsReceiver))
}

func createDefaultConfig() config.Receiver {
	return &Config{
		ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
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

func createTracesReceiver(
	_ context.Context,
	_ component.ReceiverCreateSettings,
	cfg config.Receiver,
	nextConsumer consumer.Traces,
) (component.TracesReceiver, error) {
	var err error
	r := receivers.GetOrAdd(cfg, func() component.Component {
		rCfg := cfg.(*Config)
		var recv *ocReceiver
		recv, err = newOpenCensusReceiver(rCfg.ID(), rCfg.NetAddr.Transport, rCfg.NetAddr.Endpoint, nil, nil, rCfg.buildOptions()...)
		return recv
	})
	if err != nil {
		return nil, err
	}
	r.Unwrap().(*ocReceiver).traceConsumer = nextConsumer

	return r, nil
}

func createMetricsReceiver(
	_ context.Context,
	_ component.ReceiverCreateSettings,
	cfg config.Receiver,
	nextConsumer consumer.Metrics,
) (component.MetricsReceiver, error) {
	var err error
	r := receivers.GetOrAdd(cfg, func() component.Component {
		rCfg := cfg.(*Config)
		var recv *ocReceiver
		recv, err = newOpenCensusReceiver(rCfg.ID(), rCfg.NetAddr.Transport, rCfg.NetAddr.Endpoint, nil, nil, rCfg.buildOptions()...)
		return recv
	})
	if err != nil {
		return nil, err
	}
	r.Unwrap().(*ocReceiver).metricsConsumer = nextConsumer

	return r, nil
}

// This is the map of already created OpenCensus receivers for particular configurations.
// We maintain this map because the Factory is asked trace and metric receivers separately
// when it gets CreateTracesReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one ocReceiver object per configuration.
var receivers = sharedcomponent.NewSharedComponents()
