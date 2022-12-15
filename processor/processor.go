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

package processor // import "go.opentelemetry.io/collector/processor"

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

// Traces is a processor that can consume traces.
type Traces = component.TracesProcessor //nolint:staticcheck

// Metrics is a processor that can consume metrics.
type Metrics = component.MetricsProcessor //nolint:staticcheck

// Logs is a processor that can consume logs.
type Logs = component.LogsProcessor //nolint:staticcheck

// CreateSettings is passed to Create* functions in ProcessorFactory.
type CreateSettings = component.ProcessorCreateSettings //nolint:staticcheck

// Factory is Factory interface for processors.
//
// This interface cannot be directly implemented. Implementations must
// use the NewProcessorFactory to implement it.
type Factory = component.ProcessorFactory //nolint:staticcheck

// FactoryOption apply changes to Options.
type FactoryOption = component.ProcessorFactoryOption //nolint:staticcheck

// CreateTracesFunc is the equivalent of Factory.CreateTraces().
type CreateTracesFunc = component.CreateTracesProcessorFunc //nolint:staticcheck

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics().
type CreateMetricsFunc = component.CreateMetricsProcessorFunc //nolint:staticcheck

// CreateLogsFunc is the equivalent of Factory.CreateLogs().
type CreateLogsFunc = component.CreateLogsProcessorFunc //nolint:staticcheck

// WithTraces overrides the default "error not supported" implementation for CreateTraces and the default "undefined" stability level.
var WithTraces = component.WithTracesProcessor //nolint:staticcheck

// WithMetrics overrides the default "error not supported" implementation for CreateMetrics and the default "undefined" stability level.
var WithMetrics = component.WithMetricsProcessor //nolint:staticcheck

// WithLogs overrides the default "error not supported" implementation for CreateLogs and the default "undefined" stability level.
var WithLogs = component.WithLogsProcessor //nolint:staticcheck

// NewFactory returns a Factory.
var NewFactory = component.NewProcessorFactory //nolint:staticcheck

// MakeFactoryMap takes a list of factories and returns a map with Factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
var MakeFactoryMap = component.MakeProcessorFactoryMap //nolint:staticcheck

// Builder processor is a helper struct that given a set of Configs and Factories helps with creating processors.
type Builder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]Factory
}

// NewBuilder creates a new processor.Builder to help with creating components form a set of configs and factories.
func NewBuilder(cfgs map[component.ID]component.Config, factories map[component.Type]Factory) *Builder {
	return &Builder{cfgs: cfgs, factories: factories}
}

// CreateTraces creates a Traces processor based on the settings and config.
func (b *Builder) CreateTraces(ctx context.Context, set CreateSettings, next consumer.Traces) (Traces, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("processor %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("processor factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.TracesProcessorStability())
	return f.CreateTracesProcessor(ctx, set, cfg, next)
}

// CreateMetrics creates a Metrics processor based on the settings and config.
func (b *Builder) CreateMetrics(ctx context.Context, set CreateSettings, next consumer.Metrics) (Metrics, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("processor %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("processor factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.MetricsProcessorStability())
	return f.CreateMetricsProcessor(ctx, set, cfg, next)
}

// CreateLogs creates a Logs processor based on the settings and config.
func (b *Builder) CreateLogs(ctx context.Context, set CreateSettings, next consumer.Logs) (Logs, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("processor %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("processor factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.LogsProcessorStability())
	return f.CreateLogsProcessor(ctx, set, cfg, next)
}

func (b *Builder) Factory(componentType component.Type) component.Factory {
	return b.factories[componentType]
}

// logStabilityLevel logs the stability level of a component. The log level is set to info for
// undefined, unmaintained, deprecated and development. The log level is set to debug
// for alpha, beta and stable.
func logStabilityLevel(logger *zap.Logger, sl component.StabilityLevel) {
	if sl >= component.StabilityLevelAlpha {
		logger.Debug(sl.LogMessage())
	} else {
		logger.Info(sl.LogMessage())
	}
}
