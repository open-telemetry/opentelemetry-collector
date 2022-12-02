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

package exporter // import "go.opentelemetry.io/collector/exporter"

import (
	"go.opentelemetry.io/collector/component"
)

// Traces is an exporter that can consume traces.
type Traces = component.TracesExporter //nolint:staticcheck

// Metrics is an exporter that can consume metrics.
type Metrics = component.MetricsExporter //nolint:staticcheck

// Logs is an exporter that can consume logs.
type Logs = component.LogsExporter //nolint:staticcheck

// CreateSettings configures exporter creators.
type CreateSettings = component.ExporterCreateSettings //nolint:staticcheck

// Factory is factory interface for exporters.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory = component.ExporterFactory //nolint:staticcheck

// FactoryOption apply changes to Factory.
type FactoryOption = component.ExporterFactoryOption //nolint:staticcheck

// CreateTracesFunc is the equivalent of Factory.CreateTraces.
type CreateTracesFunc = component.CreateTracesExporterFunc //nolint:staticcheck

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics.
type CreateMetricsFunc = component.CreateMetricsExporterFunc //nolint:staticcheck

// CreateLogsFunc is the equivalent of Factory.CreateLogs.
type CreateLogsFunc = component.CreateLogsExporterFunc //nolint:staticcheck

// WithTraces overrides the default "error not supported" implementation for CreateTracesExporter and the default "undefined" stability level.
var WithTraces = component.WithTracesExporter //nolint:staticcheck

// WithMetrics overrides the default "error not supported" implementation for CreateMetricsExporter and the default "undefined" stability level.
var WithMetrics = component.WithMetricsExporter //nolint:staticcheck

// WithLogs overrides the default "error not supported" implementation for CreateLogsExporter and the default "undefined" stability level.
var WithLogs = component.WithLogsExporter //nolint:staticcheck

// NewFactory returns a Factory.
var NewFactory = component.NewExporterFactory //nolint:staticcheck

// MakeFactoryMap takes a list of exporter factories and returns a map
// with factory type as keys. It returns a non-nil error when more than one factories
// have the same type.
var MakeFactoryMap = component.MakeExporterFactoryMap //nolint:staticcheck
