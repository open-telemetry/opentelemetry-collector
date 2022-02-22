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

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"go.opentelemetry.io/collector/component"
)

// Deprecated: [v0.46.0] use component.ExporterFactoryOption.
type FactoryOption = component.ExporterFactoryOption

// Deprecated: [v0.46.0] use component.ExporterCreateDefaultConfigFunc.
type CreateDefaultConfig = component.ExporterCreateDefaultConfigFunc

// Deprecated: [v0.46.0] use component.CreateTracesExporterFunc.
type CreateTracesExporter = component.CreateTracesExporterFunc

// Deprecated: [v0.46.0] use component.CreateMetricsExporterFunc.
type CreateMetricsExporter = component.CreateMetricsExporterFunc

// Deprecated: [v0.46.0] use component.CreateLogsExporterFunc.
type CreateLogsExporter = component.CreateLogsExporterFunc

// Deprecated: [v0.46.0] use component.WithTracesExporter.
var WithTraces = component.WithTracesExporter

// Deprecated: [v0.46.0] use component.WithMetricsExporter.
var WithMetrics = component.WithMetricsExporter

// Deprecated: [v0.46.0] use component.WithLogsExporter.
var WithLogs = component.WithLogsExporter

// Deprecated: [v0.46.0] use component.NewExporterFactory.
var NewFactory = component.NewExporterFactory
