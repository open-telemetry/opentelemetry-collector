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
	"go.opentelemetry.io/collector/component"
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

// MakeFactoryMap takes a list of processor factories and returns a map
// with factory type as keys. It returns a non-nil error when more than one factories
// have the same type.
var MakeFactoryMap = component.MakeProcessorFactoryMap //nolint:staticcheck
