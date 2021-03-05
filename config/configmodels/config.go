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

// Package configmodels defines the data models for entities. This file defines the
// models for configuration format. The defined entities are:
// Config (the top-level structure), Receivers, Exporters, Processors, Pipelines.
//
// Receivers, Exporters and Processors typically have common configuration settings, however
// sometimes specific implementations will have extra configuration settings.
// This requires the configuration data for these entities to be polymorphic.
//
// To satisfy these requirements we declare interfaces Receiver, Exporter, Processor,
// which define the behavior. We also provide helper structs ReceiverSettings, ExporterSettings,
// ProcessorSettings, which define the common settings and un-marshaling from config files.
//
// Specific Receivers/Exporters/Processors are expected to at the minimum implement the
// corresponding interface and if they have additional settings they must also extend
// the corresponding common settings struct (the easiest approach is to embed the common struct).
package configmodels

// Config defines the configuration for the various elements of collector or agent.
type Config struct {
	Receivers
	Exporters
	Processors
	Extensions
	Service
}

// Service defines the configurable components of the service.
type Service struct {
	// Extensions is the ordered list of extensions configured for the service.
	Extensions []string

	// Pipelines is the set of data pipelines configured for the service.
	Pipelines Pipelines
}

// Type is the component type as it is used in the config.
type Type string

// NamedEntity is a configuration entity that has a type and a name.
type NamedEntity interface {
	Type() Type
	Name() string
	SetName(name string)
}

// DataType is the data type that is supported for collection. We currently support
// collecting metrics, traces and logs, this can expand in the future.
type DataType string

// Currently supported data types. Add new data types here when new types are supported in the future.
const (
	// TracesDataType is the data type tag for traces.
	TracesDataType DataType = "traces"

	// MetricsDataType is the data type tag for metrics.
	MetricsDataType DataType = "metrics"

	// LogsDataType is the data type tag for logs.
	LogsDataType DataType = "logs"
)

// Pipeline defines a single pipeline.
type Pipeline struct {
	Name       string
	InputType  DataType
	Receivers  []string
	Processors []string
	Exporters  []string
}

// Pipelines is a map of names to Pipelines.
type Pipelines map[string]*Pipeline
