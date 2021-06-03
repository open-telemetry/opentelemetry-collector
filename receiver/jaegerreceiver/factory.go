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

package jaegerreceiver

// This file implements factory for Jaeger receiver.

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const (
	typeStr = "jaeger"

	// Protocol values.
	protoGRPC          = "grpc"
	protoThriftHTTP    = "thrift_http"
	protoThriftBinary  = "thrift_binary"
	protoThriftCompact = "thrift_compact"

	// Default endpoints to bind to.
	defaultGRPCBindEndpoint            = "0.0.0.0:14250"
	defaultHTTPBindEndpoint            = "0.0.0.0:14268"
	defaultThriftCompactBindEndpoint   = "0.0.0.0:6831"
	defaultThriftBinaryBindEndpoint    = "0.0.0.0:6832"
	defaultAgentRemoteSamplingHTTPPort = 5778
)

// NewFactory creates a new Jaeger receiver factory.
func NewFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		receiverhelper.WithTraces(createTracesReceiver))
}

// CreateDefaultConfig creates the default configuration for Jaeger receiver.
func createDefaultConfig() config.Receiver {
	return &Config{
		ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
		Protocols: Protocols{
			GRPC: &configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  defaultGRPCBindEndpoint,
					Transport: "tcp",
				},
			},
			ThriftHTTP: &confighttp.HTTPServerSettings{
				Endpoint: defaultHTTPBindEndpoint,
			},
			ThriftBinary: &ProtocolUDP{
				Endpoint:        defaultThriftBinaryBindEndpoint,
				ServerConfigUDP: DefaultServerConfigUDP(),
			},
			ThriftCompact: &ProtocolUDP{
				Endpoint:        defaultThriftCompactBindEndpoint,
				ServerConfigUDP: DefaultServerConfigUDP(),
			},
		},
	}
}

// createTracesReceiver creates a trace receiver based on provided config.
func createTracesReceiver(
	_ context.Context,
	set component.ReceiverCreateSettings,
	cfg config.Receiver,
	nextConsumer consumer.Traces,
) (component.TracesReceiver, error) {

	// Convert settings in the source config to configuration struct
	// that Jaeger receiver understands.

	rCfg := cfg.(*Config)
	remoteSamplingConfig := rCfg.RemoteSampling

	var config configuration
	// Set ports
	if rCfg.Protocols.GRPC != nil {
		config.CollectorGRPCServerSettings = *rCfg.Protocols.GRPC
		var err error
		config.CollectorGRPCPort, err = extractPortFromEndpoint(rCfg.Protocols.GRPC.NetAddr.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("unable to extract port for GRPC: %w", err)
		}
	}

	if rCfg.Protocols.ThriftHTTP != nil {
		var err error
		config.CollectorHTTPPort, err = extractPortFromEndpoint(rCfg.Protocols.ThriftHTTP.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("unable to extract port for ThriftHTTP: %w", err)
		}

		config.CollectorHTTPSettings = *rCfg.ThriftHTTP
	}

	if rCfg.Protocols.ThriftBinary != nil {
		config.AgentBinaryThriftConfig = rCfg.ThriftBinary.ServerConfigUDP
		var err error
		config.AgentBinaryThriftPort, err = extractPortFromEndpoint(rCfg.Protocols.ThriftBinary.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("unable to extract port for ThriftBinary: %w", err)
		}
	}

	if rCfg.Protocols.ThriftCompact != nil {
		config.AgentCompactThriftConfig = rCfg.ThriftCompact.ServerConfigUDP
		var err error
		config.AgentCompactThriftPort, err = extractPortFromEndpoint(rCfg.Protocols.ThriftCompact.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("unable to extract port for ThriftCompact: %w", err)
		}
	}

	if remoteSamplingConfig != nil {
		config.RemoteSamplingClientSettings = remoteSamplingConfig.GRPCClientSettings
		if len(config.RemoteSamplingClientSettings.Endpoint) == 0 {
			config.RemoteSamplingClientSettings.Endpoint = defaultGRPCBindEndpoint
		}

		if len(remoteSamplingConfig.HostEndpoint) == 0 {
			config.AgentHTTPPort = defaultAgentRemoteSamplingHTTPPort
		} else {
			var err error
			config.AgentHTTPPort, err = extractPortFromEndpoint(remoteSamplingConfig.HostEndpoint)
			if err != nil {
				return nil, err
			}
		}

		// strategies are served over grpc so if grpc is not enabled and strategies are present return an error
		if len(remoteSamplingConfig.StrategyFile) != 0 {
			if config.CollectorGRPCPort == 0 {
				return nil, fmt.Errorf("strategy file requires the GRPC protocol to be enabled")
			}

			config.RemoteSamplingStrategyFile = remoteSamplingConfig.StrategyFile
		}
	}

	if (rCfg.Protocols.GRPC == nil && rCfg.Protocols.ThriftHTTP == nil && rCfg.Protocols.ThriftBinary == nil && rCfg.Protocols.ThriftCompact == nil) ||
		(config.CollectorGRPCPort == 0 && config.CollectorHTTPPort == 0 && config.CollectorThriftPort == 0 && config.AgentBinaryThriftPort == 0 && config.AgentCompactThriftPort == 0) {
		err := fmt.Errorf("either GRPC(%v), ThriftHTTP(%v), ThriftCompact(%v), or ThriftBinary(%v) protocol endpoint with non-zero port must be enabled for %s receiver",
			rCfg.Protocols.GRPC,
			rCfg.Protocols.ThriftHTTP,
			rCfg.Protocols.ThriftCompact,
			rCfg.Protocols.ThriftBinary,
			cfg.ID().String(),
		)
		return nil, err
	}

	// Create the receiver.
	return newJaegerReceiver(rCfg.ID(), &config, nextConsumer, set), nil
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
