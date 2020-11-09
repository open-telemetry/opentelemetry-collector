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

package resourceprocessor

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/translator/conventions"
)

const (
	// The value of "type" key in configuration.
	typeStr = "resource"
)

var processorCapabilities = component.ProcessorCapabilities{MutatesConsumedData: true}

// NewFactory returns a new factory for the Resource processor.
func NewFactory() component.ProcessorFactory {
	return processorhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		processorhelper.WithTraces(createTraceProcessor),
		processorhelper.WithMetrics(createMetricsProcessor),
		processorhelper.WithLogs(createLogsProcessor))
}

// Note: This isn't a valid configuration because the processor would do no work.
func createDefaultConfig() configmodels.Processor {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

func createTraceProcessor(
	_ context.Context,
	params component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.TracesConsumer) (component.TracesProcessor, error) {
	attrProc, err := createAttrProcessor(cfg.(*Config), params.Logger)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewTraceProcessor(
		cfg,
		nextConsumer,
		&resourceProcessor{attrProc: attrProc},
		processorhelper.WithCapabilities(processorCapabilities))
}

func createMetricsProcessor(
	_ context.Context,
	params component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.MetricsConsumer) (component.MetricsProcessor, error) {
	attrProc, err := createAttrProcessor(cfg.(*Config), params.Logger)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewMetricsProcessor(
		cfg,
		nextConsumer,
		&resourceProcessor{attrProc: attrProc},
		processorhelper.WithCapabilities(processorCapabilities))
}

func createLogsProcessor(
	_ context.Context,
	params component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.LogsConsumer) (component.LogsProcessor, error) {
	attrProc, err := createAttrProcessor(cfg.(*Config), params.Logger)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewLogsProcessor(
		cfg,
		nextConsumer,
		&resourceProcessor{attrProc: attrProc},
		processorhelper.WithCapabilities(processorCapabilities))
}

func createAttrProcessor(cfg *Config, logger *zap.Logger) (*processorhelper.AttrProc, error) {
	handleDeprecatedFields(cfg, logger)
	if len(cfg.AttributesActions) == 0 {
		return nil, fmt.Errorf("error creating \"%q\" processor due to missing required field \"attributes\"", cfg.Name())
	}
	attrProc, err := processorhelper.NewAttrProc(&processorhelper.Settings{Actions: cfg.AttributesActions})
	if err != nil {
		return nil, fmt.Errorf("error creating \"%q\" processor: %w", cfg.Name(), err)
	}
	return attrProc, nil
}

// handleDeprecatedFields converts deprecated ResourceType and Labels fields into Attributes.Upsert
func handleDeprecatedFields(cfg *Config, logger *zap.Logger) {

	// Upsert value from deprecated ResourceType config to resource attributes with "opencensus.resourcetype" key
	if cfg.ResourceType != "" {
		logger.Warn("[DEPRECATED] \"type\" field is deprecated and will be removed in future release. " +
			"Please set the value to \"attributes\" with key=opencensus.resourcetype and action=upsert.")
		upsertResourceType := processorhelper.ActionKeyValue{
			Action: processorhelper.UPSERT,
			Key:    conventions.OCAttributeResourceType,
			Value:  cfg.ResourceType,
		}
		cfg.AttributesActions = append(cfg.AttributesActions, upsertResourceType)
	}

	// Upsert values from deprecated Labels config to resource attributes
	if len(cfg.Labels) > 0 {
		logger.Warn("[DEPRECATED] \"labels\" field is deprecated and will be removed in future release. " +
			"Please use \"attributes\" field instead.")
		for k, v := range cfg.Labels {
			action := processorhelper.ActionKeyValue{Action: processorhelper.UPSERT, Key: k, Value: v}
			cfg.AttributesActions = append(cfg.AttributesActions, action)
		}
	}
}
