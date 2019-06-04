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

package opencensusreceiver

import (
	"context"

	"github.com/census-instrumentation/opencensus-service/internal/configmodels"
	"github.com/census-instrumentation/opencensus-service/internal/factories"

	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/receiver"
)

var _ = factories.RegisterReceiverFactory(&receiverFactory{})

const (
	// The value of "type" key in configuration.
	typeStr = "opencensus"
)

// receiverFactory is the factory for receiver.
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

// CreateDefaultConfig creates the default configuration for receiver.
func (f *receiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &ConfigV2{
		ReceiverSettings: configmodels.ReceiverSettings{
			Endpoint: "127.0.0.1:55678",
			Enabled:  true,
		},
	}
}

// CreateTraceReceiver creates a  trace receiver based on provided config.
func (f *receiverFactory) CreateTraceReceiver(
	ctx context.Context,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {

	r, err := f.createReceiver(cfg)
	if err != nil {
		return nil, err
	}

	r.traceConsumer = nextConsumer

	return r, nil
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func (f *receiverFactory) CreateMetricsReceiver(
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {

	r, err := f.createReceiver(cfg)
	if err != nil {
		return nil, err
	}

	r.metricsConsumer = consumer

	return r, nil
}

func (f *receiverFactory) createReceiver(cfg configmodels.Receiver) (*Receiver, error) {
	rCfg := cfg.(*ConfigV2)

	// There must be one receiver for both metrics and traces. We maintain a map of
	// receivers per config.

	// Check to see if there is already a receiver for this config.
	receiver, ok := receivers[rCfg]
	if !ok {
		// We don't have a receiver, so create one.
		var err error
		receiver, err = New(rCfg.Endpoint, nil, nil)
		if err != nil {
			return nil, err
		}
		// Remember the receiver in the map
		receivers[rCfg] = receiver
	}
	return receiver, nil
}

// This is the map of already created OpenCensus receivers for particular configurations.
// We maintain this map because the factory is asked trace and metric receivers separately
// when it gets CreateTraceReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one Receiver object per configuration.
var receivers = map[*ConfigV2]*Receiver{}
