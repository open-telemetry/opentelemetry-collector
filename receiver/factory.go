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
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer"
)

// TraceReceiverFactory is an interface that builds a new TraceReceiver based on
// some viper.Viper configuration.
type TraceReceiverFactory interface {
	// Type gets the type of the TraceReceiver created by this factory.
	Type() string
	// NewFromViper takes a viper.Viper config and creates a new TraceReceiver which uses next as the
	// next TraceConsumer in the pipeline. The factory can use the logger and pass it to the receiver if
	// appropriate
	NewFromViper(v *viper.Viper, next consumer.TraceConsumer, logger *zap.Logger) (receiver TraceReceiver, err error)
	// DefaultConfig gets the default configuration for the TraceReceiver
	// created by this factory.
	DefaultConfig() interface{}
}

// MetricsReceiverFactory is an interface that builds a new MetricsReceiver based on
// some viper.Viper configuration.
type MetricsReceiverFactory interface {
	// Type gets the type of the MetricsReceiver created by this factory.
	Type() string
	// NewFromViper takes a viper.Viper config and creates a new MetricsReceiver which uses next as the
	// next MetricsConsumer in the pipeline. The factory can use the logger and pass it to the receiver if
	// appropriate.
	NewFromViper(v *viper.Viper, next consumer.MetricsConsumer, logger *zap.Logger) (receiver MetricsReceiver, err error)
	// DefaultConfig gets the default configuration for the MetricsReceiver
	// created by this factory.
	DefaultConfig() interface{}
}
