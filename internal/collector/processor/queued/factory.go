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

package queued

import (
	"time"

	"github.com/open-telemetry/opentelemetry-service/config/configerror"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/processor"
	"go.uber.org/zap"
)

var _ = processor.RegisterFactory(&factory{})

const (
	// The value of "type" key in configuration.
	typeStr = "queued-retry"
)

// factory is the factory for OpenCensus exporter.
type factory struct {
}

// Type gets the type of the Option config created by this factory.
func (f *factory) Type() string {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for exporter.
func (f *factory) CreateDefaultConfig() configmodels.Processor {
	return &ConfigV2{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		NumWorkers:     10,
		QueueSize:      5000,
		RetryOnFailure: true,
		BackoffDelay:   time.Second * 5,
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *factory) CreateTraceProcessor(
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor,
) (processor.TraceProcessor, error) {
	oCfg := cfg.(*ConfigV2)
	return NewQueuedSpanProcessor(nextConsumer,
		Options.WithNumWorkers(oCfg.NumWorkers),
		Options.WithQueueSize(oCfg.QueueSize),
		Options.WithRetryOnProcessingFailures(oCfg.RetryOnFailure),
		Options.WithBackoffDelay(oCfg.BackoffDelay),
	), nil
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *factory) CreateMetricsProcessor(
	logger *zap.Logger,
	nextConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (processor.MetricsProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}
