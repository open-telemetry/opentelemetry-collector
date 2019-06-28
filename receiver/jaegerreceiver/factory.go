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
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/factories"
	"github.com/open-telemetry/opentelemetry-service/models"
	"github.com/open-telemetry/opentelemetry-service/receiver"
)

var _ = factories.RegisterReceiverFactory(&receiverFactory{})

const (
	// The value of "type" key in configuration.
	typeStr = "jaeger"

	// Protocol values.
	protoThriftHTTP     = "thrift-http"
	protoThriftTChannel = "thrift-tchannel"

	// Default endpoints to bind to.
	defaultHTTPBindEndpoint     = "127.0.0.1:14268"
	defaultTChannelBindEndpoint = "127.0.0.1:14267"
)

// receiverFactory is the factory for Jaeger receiver.
type receiverFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *receiverFactory) Type() string {
	return typeStr
}

// CustomUnmarshaler returns nil because we don't need custom unmarshaling for this config.
func (f *receiverFactory) CustomUnmarshaler() factories.CustomUnmarshaler {
	return nil
}

// CreateDefaultConfig creates the default configuration for Jaeger receiver.
func (f *receiverFactory) CreateDefaultConfig() models.Receiver {
	return &ConfigV2{
		TypeVal: typeStr,
		NameVal: typeStr,
		Protocols: map[string]*models.ReceiverSettings{
			protoThriftTChannel: {
				Enabled:  false,
				Endpoint: defaultTChannelBindEndpoint,
			},
			protoThriftHTTP: {
				Enabled:  false,
				Endpoint: defaultHTTPBindEndpoint,
			},
		},
	}
}

// CreateTraceReceiver creates a trace receiver based on provided config.
func (f *receiverFactory) CreateTraceReceiver(
	ctx context.Context,
	cfg models.Receiver,
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {

	// Convert settings in the source config to Configuration struct
	// that Jaeger receiver understands.

	rCfg := cfg.(*ConfigV2)

	protoHTTP := rCfg.Protocols[protoThriftHTTP]
	protoTChannel := rCfg.Protocols[protoThriftTChannel]

	config := Configuration{}

	// Set ports
	if protoHTTP != nil {
		var err error
		config.CollectorHTTPPort, err = extractPortFromEndpoint(protoHTTP.Endpoint)
		if err != nil {
			return nil, err
		}
	}

	if protoTChannel != nil {
		var err error
		config.CollectorThriftPort, err = extractPortFromEndpoint(protoTChannel.Endpoint)
		if err != nil {
			return nil, err
		}
	}

	if (protoHTTP == nil && protoTChannel == nil) ||
		(config.CollectorHTTPPort == 0 && config.CollectorThriftPort == 0) {
		return nil, errors.New("either " + protoThriftHTTP + " or " + protoThriftTChannel +
			" protocol endpoint with non-zero port must be defined for " + typeStr + " receiver")
	}

	// Jaeger receiver implementation currently does not allow specifying which interface
	// to bind to so we cannot use yet the address part of endpoint.

	// Create the receiver.
	return New(ctx, &config, nextConsumer)
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func (f *receiverFactory) CreateMetricsReceiver(
	cfg models.Receiver,
	consumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {
	return nil, factories.ErrDataTypeIsNotSupported
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
