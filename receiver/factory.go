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

package receiver

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
)

// Factory is factory interface for receivers.
type Factory interface {
	// Type gets the type of the Receiver created by this factory.
	Type() string

	// CreateDefaultConfig creates the default configuration for the Receiver.
	CreateDefaultConfig() configmodels.Receiver

	// CustomUnmarshaler returns a custom unmarshaler for the configuration or nil if
	// there is no need for custom unmarshaling. This is typically used if viper.Unmarshal()
	// is not sufficient to unmarshal correctly.
	CustomUnmarshaler() CustomUnmarshaler

	// CreateTraceReceiver creates a trace receiver based on this config.
	// If the receiver type does not support tracing or if the config is not valid
	// error will be returned instead.
	CreateTraceReceiver(ctx context.Context, logger *zap.Logger, cfg configmodels.Receiver,
		nextConsumer consumer.TraceConsumer) (TraceReceiver, error)

	// CreateMetricsReceiver creates a metrics receiver based on this config.
	// If the receiver type does not support metrics or if the config is not valid
	// error will be returned instead.
	CreateMetricsReceiver(logger *zap.Logger, cfg configmodels.Receiver,
		consumer consumer.MetricsConsumer) (MetricsReceiver, error)
}

// CustomUnmarshaler is a function that un-marshals a viper data into a config struct
// in a custom way.
type CustomUnmarshaler func(v *viper.Viper, viperKey string, intoCfg interface{}) error

// List of registered receiver factories.
var receiverFactories = make(map[string]Factory)

// RegisterFactory registers a receiver factory.
func RegisterFactory(factory Factory) error {
	if receiverFactories[factory.Type()] != nil {
		panic(fmt.Sprintf("duplicate receiver factory %q", factory.Type()))
	}

	receiverFactories[factory.Type()] = factory
	return nil
}

// GetFactory gets a receiver factory by type string.
func GetFactory(typeStr string) Factory {
	return receiverFactories[typeStr]
}
