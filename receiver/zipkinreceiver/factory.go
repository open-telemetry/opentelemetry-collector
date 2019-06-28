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

package zipkinreceiver

import (
	"context"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/factories"
	"github.com/open-telemetry/opentelemetry-service/models"
	"github.com/open-telemetry/opentelemetry-service/receiver"
)

// This file implements factory for Zipkin receiver.

var _ = factories.RegisterReceiverFactory(&ReceiverFactory{})

const (
	// The value of "type" key in configuration.
	typeStr = "zipkin"

	defaultBindEndpoint = "127.0.0.1:9411"
)

// ReceiverFactory is the factory for Zipkin receiver.
type ReceiverFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *ReceiverFactory) Type() string {
	return typeStr
}

// CustomUnmarshaler returns nil because we don't need custom unmarshaling for this config.
func (f *ReceiverFactory) CustomUnmarshaler() factories.CustomUnmarshaler {
	return nil
}

// CreateDefaultConfig creates the default configuration for Jaeger receiver.
func (f *ReceiverFactory) CreateDefaultConfig() models.Receiver {
	return &ConfigV2{
		ReceiverSettings: models.ReceiverSettings{
			TypeVal:  typeStr,
			NameVal:  typeStr,
			Endpoint: defaultBindEndpoint,
		},
	}
}

// CreateTraceReceiver creates a trace receiver based on provided config.
func (f *ReceiverFactory) CreateTraceReceiver(
	ctx context.Context,
	cfg models.Receiver,
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {

	rCfg := cfg.(*ConfigV2)
	return New(rCfg.Endpoint, nextConsumer)
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func (f *ReceiverFactory) CreateMetricsReceiver(
	cfg models.Receiver,
	consumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {
	return nil, factories.ErrDataTypeIsNotSupported
}
