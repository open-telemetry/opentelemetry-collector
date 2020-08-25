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

package otlpreceiver

import (
	"context"
	"fmt"

	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr = "otlp"

	// Protocol values.
	protoGRPC          = "grpc"
	protoHTTP          = "http"
	protocolsFieldName = "protocols"
)

func NewFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		receiverhelper.WithTraces(createTraceReceiver),
		receiverhelper.WithMetrics(createMetricsReceiver),
		receiverhelper.WithLogs(createLogReceiver),
		receiverhelper.WithCustomUnmarshaler(customUnmarshaler))
}

// createDefaultConfig creates the default configuration for receiver.
func createDefaultConfig() configmodels.Receiver {
	return &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		Protocols: Protocols{
			GRPC: &configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  "0.0.0.0:55680",
					Transport: "tcp",
				},
				// We almost write 0 bytes, so no need to tune WriteBufferSize.
				ReadBufferSize: 512 * 1024,
			},
			HTTP: &confighttp.HTTPServerSettings{
				Endpoint: "0.0.0.0:55681",
			},
		},
	}
}

// customUnmarshaler is used to add defaults for named but empty protocols
func customUnmarshaler(componentViperSection *viper.Viper, intoCfg interface{}) error {
	if componentViperSection == nil || len(componentViperSection.AllKeys()) == 0 {
		return fmt.Errorf("empty config for OTLP receiver")
	}
	// first load the config normally
	err := componentViperSection.UnmarshalExact(intoCfg)
	if err != nil {
		return err
	}

	receiverCfg := intoCfg.(*Config)
	// next manually search for protocols in viper, if a protocol is not present it means it is disable.
	protocols := componentViperSection.GetStringMap(protocolsFieldName)

	// UnmarshalExact will ignore empty entries like a protocol with no values, so if a typo happened
	// in the protocol that is intended to be enabled will not be enabled. So check if the protocols
	// include only known protocols.
	knownProtocols := 0
	if _, ok := protocols[protoGRPC]; !ok {
		receiverCfg.GRPC = nil
	} else {
		knownProtocols++
	}

	if _, ok := protocols[protoHTTP]; !ok {
		receiverCfg.HTTP = nil
	} else {
		knownProtocols++
	}

	if len(protocols) != knownProtocols {
		return fmt.Errorf("unknown protocols in the OTLP receiver")
	}

	if receiverCfg.GRPC == nil && receiverCfg.HTTP == nil {
		return fmt.Errorf("must specify at least one protocol when using the OTLP receiver")
	}

	return nil
}

// CreateTraceReceiver creates a  trace receiver based on provided config.
func createTraceReceiver(
	ctx context.Context,
	_ component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer,
) (component.TraceReceiver, error) {
	r, err := createReceiver(cfg)
	if err != nil {
		return nil, err
	}
	if err = r.registerTraceConsumer(ctx, nextConsumer); err != nil {
		return nil, err
	}
	return r, nil
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func createMetricsReceiver(
	ctx context.Context,
	_ component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (component.MetricsReceiver, error) {
	r, err := createReceiver(cfg)
	if err != nil {
		return nil, err
	}
	if err = r.registerMetricsConsumer(ctx, consumer); err != nil {
		return nil, err
	}
	return r, nil
}

// CreateLogReceiver creates a log receiver based on provided config.
func createLogReceiver(
	ctx context.Context,
	_ component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	consumer consumer.LogsConsumer,
) (component.LogsReceiver, error) {
	r, err := createReceiver(cfg)
	if err != nil {
		return nil, err
	}
	if err = r.registerLogsConsumer(ctx, consumer); err != nil {
		return nil, err
	}
	return r, nil
}

func createReceiver(cfg configmodels.Receiver) (*otlpReceiver, error) {
	rCfg := cfg.(*Config)

	// There must be one receiver for both metrics and traces. We maintain a map of
	// receivers per config.

	// Check to see if there is already a receiver for this config.
	receiver, ok := receivers[rCfg]
	if !ok {
		var err error
		// We don't have a receiver, so create one.
		receiver, err = newOtlpReceiver(rCfg)
		if err != nil {
			return nil, err
		}
		// Remember the receiver in the map
		receivers[rCfg] = receiver
	}
	return receiver, nil
}

// This is the map of already created OTLP receivers for particular configurations.
// We maintain this map because the Factory is asked trace and metric receivers separately
// when it gets CreateTraceReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one otlpReceiver object per configuration.
var receivers = map[*Config]*otlpReceiver{}
