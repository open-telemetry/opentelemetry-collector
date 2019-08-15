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

package span

import (
	"errors"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/config/configerror"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/processor"
)

const (
	// typeStr is the value of "type" Span Rename in the configuration.
	typeStr = "span"
)

// errMissingRequiredField is returned when a required field in the config
// is not specified.
var errMissingRequiredField = errors.New("error creating \"span\" processor due to missing required field \"keys\"")

// Factory is the factory for the Span Rename processor.
type Factory struct {
}

// Type gets the type of the config created by this factory.
func (f *Factory) Type() string {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for processor.
func (f *Factory) CreateDefaultConfig() configmodels.Processor {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
// TODO(ccaraman): Use NewTraceProcessor when added in follow up PR.
func (f *Factory) CreateTraceProcessor(
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor) (processor.TraceProcessor, error) {

	// Keys has to be set for the span-rename processor to be valid.
	// If not set and not enforced, the processor would do no work.
	oCfg := cfg.(*Config)
	if len(oCfg.Rename.Keys) == 0 {
		return nil, errMissingRequiredField
	}

	// TODO(ccaraman): Returning an error for first pr that only implements the
	// 	config. Follow up PR will add functionality and replace this error with a
	// 	call to instantiate a new TraceProcessor.
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsProcessor creates a metric processor based on this config.
func (f *Factory) CreateMetricsProcessor(
	logger *zap.Logger,
	nextConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor) (processor.MetricsProcessor, error) {
	// Span Rename Processor does not support Metrics.
	return nil, configerror.ErrDataTypeIsNotSupported
}
