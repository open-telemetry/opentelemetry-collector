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

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/components"
	"go.opentelemetry.io/collector/service/internal/fanoutconsumer"
)

const (
	zPipelineName  = "zpipelinename"
	zComponentName = "zcomponentname"
	zComponentKind = "zcomponentkind"
)

// baseConsumer redeclared here since not public in consumer package. May consider to make that public.
type baseConsumer interface {
	Capabilities() consumer.Capabilities
}

// pipelinesSettings holds configuration for building builtPipelines.
type pipelinesSettings struct {
	Telemetry component.TelemetrySettings
	BuildInfo component.BuildInfo

	ReceiverBuilder  *receiver.Builder
	ProcessorBuilder *processor.Builder
	ExporterBuilder  *exporter.Builder
	ConnectorBuilder *connector.Builder

	// PipelineConfigs is a map of component.ID to PipelineConfig.
	PipelineConfigs map[component.ID]*PipelineConfig
}

func buildExporter(
	ctx context.Context,
	componentID component.ID,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *exporter.Builder,
	pipelineID component.ID,
) (exp component.Component, err error) {
	set := exporter.CreateSettings{ID: componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ExporterLogger(set.TelemetrySettings.Logger, componentID, pipelineID.Type())
	switch pipelineID.Type() {
	case component.DataTypeTraces:
		exp, err = builder.CreateTraces(ctx, set)
	case component.DataTypeMetrics:
		exp, err = builder.CreateMetrics(ctx, set)
	case component.DataTypeLogs:
		exp, err = builder.CreateLogs(ctx, set)
	default:
		return nil, fmt.Errorf("error creating exporter %q in pipeline %q, data type %q is not supported", set.ID, pipelineID, pipelineID.Type())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create %q exporter, in pipeline %q: %w", set.ID, pipelineID, err)
	}
	return exp, nil
}

func buildProcessor(ctx context.Context,
	componentID component.ID,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *processor.Builder,
	pipelineID component.ID,
	next baseConsumer,
) (proc component.Component, err error) {
	set := processor.CreateSettings{ID: componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ProcessorLogger(set.TelemetrySettings.Logger, componentID, pipelineID)
	switch pipelineID.Type() {
	case component.DataTypeTraces:
		proc, err = builder.CreateTraces(ctx, set, next.(consumer.Traces))
	case component.DataTypeMetrics:
		proc, err = builder.CreateMetrics(ctx, set, next.(consumer.Metrics))
	case component.DataTypeLogs:
		proc, err = builder.CreateLogs(ctx, set, next.(consumer.Logs))
	default:
		return nil, fmt.Errorf("error creating processor %q in pipeline %q, data type %q is not supported", set.ID, pipelineID, pipelineID.Type())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, pipelineID, err)
	}
	return proc, nil
}

func buildReceiver(ctx context.Context,
	componentID component.ID,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *receiver.Builder,
	pipelineID component.ID,
	nexts []baseConsumer,
) (recv component.Component, err error) {
	set := receiver.CreateSettings{ID: componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ReceiverLogger(tel.Logger, componentID, pipelineID.Type())
	switch pipelineID.Type() {
	case component.DataTypeTraces:
		var consumers []consumer.Traces
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Traces))
		}
		recv, err = builder.CreateTraces(ctx, set, fanoutconsumer.NewTraces(consumers))
	case component.DataTypeMetrics:
		var consumers []consumer.Metrics
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Metrics))
		}
		recv, err = builder.CreateMetrics(ctx, set, fanoutconsumer.NewMetrics(consumers))
	case component.DataTypeLogs:
		var consumers []consumer.Logs
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Logs))
		}
		recv, err = builder.CreateLogs(ctx, set, fanoutconsumer.NewLogs(consumers))
	default:
		return nil, fmt.Errorf("error creating receiver %q in pipeline %q, data type %q is not supported", set.ID, pipelineID, pipelineID.Type())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create %q receiver, in pipeline %q: %w", set.ID, pipelineID, err)
	}
	return recv, nil
}
