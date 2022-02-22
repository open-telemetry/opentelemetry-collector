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

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"go.opentelemetry.io/collector/component"
)

// Deprecated: [v0.46.0] use component.ProcessorFactoryOption.
type FactoryOption = component.ProcessorFactoryOption

// Deprecated: [v0.46.0] use component.ProcessorCreateDefaultConfigFunc.
type CreateDefaultConfig = component.ProcessorCreateDefaultConfigFunc

// Deprecated: [v0.46.0] use component.CreateTracesProcessorFunc.
type CreateTracesProcessor = component.CreateTracesProcessorFunc

// Deprecated: [v0.46.0] use component.CreateMetricsProcessorFunc.
type CreateMetricsProcessor = component.CreateMetricsProcessorFunc

// Deprecated: [v0.46.0] use component.CreateLogsProcessorFunc.
type CreateLogsProcessor = component.CreateLogsProcessorFunc

// Deprecated: [v0.46.0] use component.WithTracesProcessor.
var WithTraces = component.WithTracesProcessor

// Deprecated: [v0.46.0] use component.WithMetricsProcessor.
var WithMetrics = component.WithMetricsProcessor

// Deprecated: [v0.46.0] use component.WithLogsProcessor.
var WithLogs = component.WithLogsProcessor

// Deprecated: [v0.46.0] use component.NewProcessorFactory.
var NewFactory = component.NewProcessorFactory
