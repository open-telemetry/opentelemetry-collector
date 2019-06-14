// Copyright 2019, OpenCensus Authors
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

// Package configmodels defines the data models for entities. This file defines the
// models for V2 configuration format. The defined entities are:
// ConfigV2 (the top-level structure), Receivers, Exporters, Processors, Pipelines.
package configmodels

/*
Receivers, Exporters and Processors typically have common configuration settings, however
sometimes specific implementations (e.g. StackDriver) will have extra configuration settings.
This requires the configuration data for these entities to be polymorphic.

To satisfy these requirements we declare interfaces Receiver, Exporter, Processor,
which define the behavior. We also provide helper structs ReceiverSettings, ExporterSettings,
ProcessorSettings, which define the common settings and un-marshaling from config files.

Specific Receivers/Exporters/Processors are expected to at the minimum implement the
corresponding interface and if they have additional settings they must also extend
the corresponding common settings struct (the easiest approach is to embed the common struct).
*/

// ConfigV2 defines the configuration V2 for the various elements of collector or agent.
type ConfigV2 struct {
	Receivers  Receivers
	Exporters  Exporters
	Processors Processors
	Pipelines  Pipelines
}

// Receiver is the configuration of a receiver. Specific receivers must implement this
// interface and will typically embed ReceiverSettings struct or a struct that extends it.
type Receiver interface {
}

// Receivers is a map of names to Receivers.
type Receivers map[string]Receiver

// Exporter is the configuration of an exporter.
type Exporter interface {
	Name() string
	SetName(name string)

	Type() string
	SetType(typeStr string)
}

// Exporters is a map of names to Exporters.
type Exporters map[string]Exporter

// Processor is the configuration of a processor. Specific processors must implement this
// interface and will typically embed ProcessorSettings struct or a struct that extends it.
type Processor interface {
	Type() string
	SetType(typeStr string)
}

// Processors is a map of names to Processors.
type Processors map[string]Processor

// DataType is the data type that is supported for collection. We currently support
// collecting metrics and traces, this can expand in the future (e.g. logs, events, etc).
type DataType int

// Currently supported data types. Add new data types here when new types are supported in the future.
const (
	_ DataType = iota // skip 0, start types from 1.

	// TracesDataType is the data type tag for traces.
	TracesDataType

	// MetricsDataType is the data type tag for metrics.
	MetricsDataType
)

// Data type strings.
const (
	TracesDataTypeStr  = "traces"
	MetricsDataTypeStr = "metrics"
)

// GetDataTypeStr converts data type to string.
func (dataType DataType) GetDataTypeStr() string {
	switch dataType {
	case TracesDataType:
		return TracesDataTypeStr
	case MetricsDataType:
		return MetricsDataTypeStr
	default:
		panic("unknown data type")
	}
}

// Pipeline defines a single pipeline.
type Pipeline struct {
	Name       string   `mapstructure:"-"`
	InputType  DataType `mapstructure:"-"`
	Receivers  []string `mapstructure:"receivers"`
	Processors []string `mapstructure:"processors"`
	Exporters  []string `mapstructure:"exporters"`
}

// Pipelines is a map of names to Pipelines.
type Pipelines map[string]*Pipeline

// Below are common setting structs for Receivers, Exporters and Processors.
// These are helper structs which you can embed when implementing your specific
// receiver/exporter/processor config storage.

// ReceiverSettings defines common settings for a single-protocol receiver configuration.
// Specific receivers can embed this struct and extend it with more fields if needed.
type ReceiverSettings struct {
	Enabled  bool   `mapstructure:"enabled"`
	Endpoint string `mapstructure:"endpoint"`
}

// ExporterSettings defines common settings for an exporter configuration.
// Specific exporters can embed this struct and extend it with more fields if needed.
type ExporterSettings struct {
	TypeVal string `mapstructure:"-"`
	NameVal string `mapstructure:"-"`
	Enabled bool   `mapstructure:"enabled"`
}

var _ Exporter = (*ExporterSettings)(nil)

// Name gets the exporter name.
func (es *ExporterSettings) Name() string {
	return es.NameVal
}

// SetName sets the exporter name.
func (es *ExporterSettings) SetName(name string) {
	es.NameVal = name
}

// Type sets the exporter type.
func (es *ExporterSettings) Type() string {
	return es.TypeVal
}

// SetType sets the exporter type.
func (es *ExporterSettings) SetType(typeStr string) {
	es.TypeVal = typeStr
}

// ProcessorSettings defines common settings for a processor configuration.
// Specific processors can embed this struct and extend it with more fields if needed.
type ProcessorSettings struct {
	TypeVal string `mapstructure:"-"`
	Enabled bool   `mapstructure:"enabled"`
}

// Type sets the processor type.
func (proc *ProcessorSettings) Type() string {
	return proc.TypeVal
}

// SetType sets the processor type.
func (proc *ProcessorSettings) SetType(typeStr string) {
	proc.TypeVal = typeStr
}

var _ Processor = (*ProcessorSettings)(nil)
