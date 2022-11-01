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

package config // import "go.opentelemetry.io/collector/config"

import (
	"go.opentelemetry.io/collector/component"
)

// Pipeline defines a single pipeline.
// Deprecated: [v0.52.0] Use service.ConfigServicePipeline
type Pipeline struct {
	Receivers  []component.ID `mapstructure:"receivers"`
	Processors []component.ID `mapstructure:"processors"`
	Exporters  []component.ID `mapstructure:"exporters"`
}

// Deprecated: [v0.52.0] will be removed soon.
type Pipelines = map[component.ID]*Pipeline

// Deprecated: [v0.64.0] use component.ReceiverConfig.
type Receiver = component.ReceiverConfig

// Deprecated: [v0.64.0] use component.UnmarshalReceiverConfig.
var UnmarshalReceiver = component.UnmarshalReceiverConfig

// Deprecated: [v0.64.0] use component.ProcessorConfig.
type Processor = component.ProcessorConfig

// Deprecated: [v0.64.0] use component.UnmarshalProcessorConfig.
var UnmarshalProcessor = component.UnmarshalProcessorConfig

// Deprecated: [v0.64.0] use component.ExporterConfig.
type Exporter = component.ExporterConfig

// Deprecated: [v0.64.0] use component.UnmarshalExporterConfig.
var UnmarshalExporter = component.UnmarshalExporterConfig

// Deprecated: [v0.64.0] use component.ExtensionConfig.
type Extension = component.ExtensionConfig

// Deprecated: [v0.64.0] use component.UnmarshalExtensionConfig.
var UnmarshalExtension = component.UnmarshalExtensionConfig

// Deprecated: [v0.64.0] use component.Type.
type Type = component.Type

// Deprecated: [v0.64.0] use component.DataType.
type DataType = component.DataType

// Deprecated: [v0.64.0] use component.TracesDataType.
const TracesDataType = component.TracesDataType

// Deprecated: [v0.64.0] use component.MetricsDataType.
const MetricsDataType = component.MetricsDataType

// Deprecated: [v0.64.0] use component.LogsDataType.
const LogsDataType = component.LogsDataType

// Deprecated: [v0.64.0] use component.ID.
type ComponentID = component.ID

// Deprecated: [v0.64.0] use component.NewID.
var NewComponentID = component.NewID

// Deprecated: [v0.64.0] use component.NewIDWithName.
var NewComponentIDWithName = component.NewIDWithName

// Deprecated: [v0.64.0] use component.ID.UnmarshalText.
func NewComponentIDFromString(idStr string) (ComponentID, error) {
	id := component.ID{}
	return id, id.UnmarshalText([]byte(idStr))
}
