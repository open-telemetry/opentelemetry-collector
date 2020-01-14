// Copyright 2019, OpenTelemetry Authors
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

package jaegerreceiver

// This file implements factory for Jaeger receiver.

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/receiver"
)

const (
	// The value of "type" key in configuration.
	typeStr = "jaeger"

	// Protocol values.
	protoGRPC       = "grpc"
	protoThriftHTTP = "thrift-http"
	// TODO https://github.com/open-telemetry/opentelemetry-collector/issues/267
	//	Remove ThriftTChannel support.
	protoThriftTChannel = "thrift-tchannel"
	protoThriftBinary   = "thrift-binary"
	protoThriftCompact  = "thrift-compact"

	// Default endpoints to bind to.
	defaultGRPCBindEndpoint     = "localhost:14250"
	defaultHTTPBindEndpoint     = "localhost:14268"
	defaultTChannelBindEndpoint = "localhost:14267"

	defaultThriftCompactBindEndpoint = "localhost:6831"
	defaultThriftBinaryBindEndpoint  = "localhost:6832"
)

// Factory is the factory for Jaeger receiver.
type Factory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *Factory) Type() string {
	return typeStr
}

// CustomUnmarshaler is used to add defaults for named but empty protocols
func (f *Factory) CustomUnmarshaler() receiver.CustomUnmarshaler {
	return func(v *viper.Viper, viperKey string, sourceViperSection *viper.Viper, intoCfg interface{}) error {
		// first load the config normally
		err := sourceViperSection.UnmarshalExact(intoCfg)
		if err != nil {
			return err
		}

		receiverCfg, ok := intoCfg.(*Config)
		if !ok {
			return fmt.Errorf("config type not *jaegerreceiver.Config")
		}

		// next manually search for protocols in viper that do not appear in the normally loaded config
		// these protocols were excluded during normal loading and we need to add defaults for them
		vSub := v.Sub(viperKey)
		if vSub == nil {
			return fmt.Errorf("Jaeger receiver config is empty")
		}
		protocols := vSub.GetStringMap(protocolsFieldName)
		if len(protocols) == 0 {
			return fmt.Errorf("must specify at least one protocol when using the Jaeger receiver")
		}
		for k := range protocols {
			if _, ok := receiverCfg.Protocols[k]; !ok {
				if receiverCfg.Protocols[k], err = defaultsForProtocol(k); err != nil {
					return err
				}
			}
		}

		return nil
	}
}

// CreateDefaultConfig creates the default configuration for Jaeger receiver.
func (f *Factory) CreateDefaultConfig() configmodels.Receiver {
	return &Config{
		TypeVal:   typeStr,
		NameVal:   typeStr,
		Protocols: map[string]*receiver.SecureReceiverSettings{},
	}
}

// CreateTraceReceiver creates a trace receiver based on provided config.
func (f *Factory) CreateTraceReceiver(
	ctx context.Context,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {

	// Convert settings in the source config to Configuration struct
	// that Jaeger receiver understands.

	rCfg := cfg.(*Config)

	protoGRPC := rCfg.Protocols[protoGRPC]
	protoHTTP := rCfg.Protocols[protoThriftHTTP]
	protoTChannel := rCfg.Protocols[protoThriftTChannel]
	protoThriftCompact := rCfg.Protocols[protoThriftCompact]
	protoThriftBinary := rCfg.Protocols[protoThriftBinary]
	remoteSamplingConfig := rCfg.RemoteSampling

	config := Configuration{}
	var grpcServerOptions []grpc.ServerOption

	// Set ports
	if protoGRPC != nil && protoGRPC.IsEnabled() {
		var err error
		config.CollectorGRPCPort, err = extractPortFromEndpoint(protoGRPC.Endpoint)
		if err != nil {
			return nil, err
		}

		if protoGRPC.TLSCredentials != nil {
			option, err := protoGRPC.TLSCredentials.ToGrpcServerOption()
			if err != nil {
				return nil, fmt.Errorf("failed to configure TLS: %v", err)
			}
			grpcServerOptions = append(grpcServerOptions, option)
		}
		config.CollectorGRPCOptions = grpcServerOptions
	}

	if protoHTTP != nil && protoHTTP.IsEnabled() {
		var err error
		config.CollectorHTTPPort, err = extractPortFromEndpoint(protoHTTP.Endpoint)
		if err != nil {
			return nil, err
		}
	}

	if protoTChannel != nil && protoTChannel.IsEnabled() {
		var err error
		config.CollectorThriftPort, err = extractPortFromEndpoint(protoTChannel.Endpoint)
		if err != nil {
			return nil, err
		}
	}

	if protoThriftBinary != nil && protoThriftBinary.IsEnabled() {
		var err error
		config.AgentBinaryThriftPort, err = extractPortFromEndpoint(protoThriftBinary.Endpoint)
		if err != nil {
			return nil, err
		}
	}

	if protoThriftCompact != nil && protoThriftCompact.IsEnabled() {
		var err error
		config.AgentCompactThriftPort, err = extractPortFromEndpoint(protoThriftCompact.Endpoint)
		if err != nil {
			return nil, err
		}
	}

	if remoteSamplingConfig != nil {
		config.RemoteSamplingEndpoint = remoteSamplingConfig.FetchEndpoint
	}

	if (protoGRPC == nil && protoHTTP == nil && protoTChannel == nil && protoThriftBinary == nil && protoThriftCompact == nil) ||
		(config.CollectorGRPCPort == 0 && config.CollectorHTTPPort == 0 && config.CollectorThriftPort == 0 && config.AgentBinaryThriftPort == 0 && config.AgentCompactThriftPort == 0) {
		err := fmt.Errorf("either %v, %v, %v, %v, or %v protocol endpoint with non-zero port must be enabled for %s receiver",
			protoGRPC,
			protoThriftHTTP,
			protoThriftTChannel,
			protoThriftCompact,
			protoThriftBinary,
			typeStr,
		)
		return nil, err
	}

	// Create the receiver.
	return New(ctx, &config, nextConsumer, logger)
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func (f *Factory) CreateMetricsReceiver(
	logger *zap.Logger,
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

// extract the port number from string in "address:port" format. If the
// port number cannot be extracted returns an error.
func extractPortFromEndpoint(endpoint string) (int, error) {
	_, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		return 0, fmt.Errorf("endpoint is not formatted correctly: %s", err.Error())
	}
	port, err := strconv.ParseInt(portStr, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("endpoint port is not a number: %s", err.Error())
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port number must be between 1 and 65535")
	}
	return int(port), nil
}

// returns a default value for a protocol name.  this really just boils down to the endpoint
func defaultsForProtocol(proto string) (*receiver.SecureReceiverSettings, error) {
	var defaultEndpoint string

	switch proto {
	case protoGRPC:
		defaultEndpoint = defaultGRPCBindEndpoint
	case protoThriftHTTP:
		defaultEndpoint = defaultHTTPBindEndpoint
	case protoThriftTChannel:
		defaultEndpoint = defaultTChannelBindEndpoint
	case protoThriftBinary:
		defaultEndpoint = defaultThriftBinaryBindEndpoint
	case protoThriftCompact:
		defaultEndpoint = defaultThriftCompactBindEndpoint
	default:
		return nil, fmt.Errorf("unknown Jaeger protocol %s", proto)
	}

	return &receiver.SecureReceiverSettings{
		ReceiverSettings: configmodels.ReceiverSettings{
			Endpoint: defaultEndpoint,
		},
	}, nil
}
