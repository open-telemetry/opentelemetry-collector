// Copyright The OpenTelemetry Authors
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
// models for configuration format. The defined entities are:
// Config (the top-level structure), Receivers, Exporters, Processors, Pipelines.
package configmodels

import (
	"errors"
	"strings"
)

// typeAndNameSeparator is the separator that is used between type and name in type/name composite keys.
const typeAndNameSeparator = "/"

/*
Receivers, Exporters and Processors typically have common configuration settings, however
sometimes specific implementations will have extra configuration settings.
This requires the configuration data for these entities to be polymorphic.

To satisfy these requirements we declare interfaces Receiver, Exporter, Processor,
which define the behavior. We also provide helper structs ReceiverSettings, ExporterSettings,
ProcessorSettings, which define the common settings and un-marshaling from config files.

Specific Receivers/Exporters/Processors are expected to at the minimum implement the
corresponding interface and if they have additional settings they must also extend
the corresponding common settings struct (the easiest approach is to embed the common struct).
*/

// Config defines the configuration for the various elements of collector or agent.
type Config struct {
	Receivers  Receivers
	Exporters  Exporters
	Processors Processors
	Extensions Extensions
	Service    Service
}

// Type is the component type as it is used in the config.
type Type string

// EntityName is a component type and an optional name.
type EntityName struct {
	Type Type
	Name string
}

// NewEntityName decodes a key in type[/name] format into Type and Name.
// Type and Name components have spaces trimmed.
// The "type" part must be present, the forward slash and "name" are optional. typeStr
// will be non-empty if err is nil.
func NewEntityName(key string) (entity EntityName, err error) {
	items := strings.SplitN(key, typeAndNameSeparator, 2)
	var typeStr Type

	if len(items) >= 1 {
		typeStr = Type(strings.TrimSpace(items[0]))
	}

	if len(items) == 0 || typeStr == "" {
		err = errors.New("type/name key must have the type part")
		return
	}

	var nameSuffix string
	if len(items) > 1 {
		// "name" part is present.
		nameSuffix = strings.TrimSpace(items[1])
		if nameSuffix == "" {
			err = errors.New("name part must be specified after " + typeAndNameSeparator + " in type/name key")
			return
		}
	} else {
		nameSuffix = ""
	}

	return EntityName{
		Type: typeStr,
		Name: nameSuffix,
	}, nil
}

func (e EntityName) String() string {
	if e.Name == "" {
		return string(e.Type)
	} else {
		return string(e.Type) + typeAndNameSeparator + e.Name
	}
}

// NamedEntity is a configuration entity that has a type and a name.
type NamedEntity interface {
	Name() EntityName
	SetName(name EntityName)
}

// Receiver is the configuration of a receiver. Specific receivers must implement this
// interface and will typically embed ReceiverSettings struct or a struct that extends it.
type Receiver interface {
	NamedEntity
}

// Receivers is a map of names to Receivers.
type Receivers map[string]Receiver

// Exporter is the configuration of an exporter.
type Exporter interface {
	NamedEntity
}

// Exporters is a map of names to Exporters.
type Exporters map[string]Exporter

// Processor is the configuration of a processor. Specific processors must implement this
// interface and will typically embed ProcessorSettings struct or a struct that extends it.
type Processor interface {
	NamedEntity
}

// Processors is a map of names to Processors.
type Processors map[string]Processor

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
	Name       string   `mapstructure:"-"`
	InputType  DataType `mapstructure:"-"`
	Receivers  []string `mapstructure:"receivers"`
	Processors []string `mapstructure:"processors"`
	Exporters  []string `mapstructure:"exporters"`
}

// Pipelines is a map of names to Pipelines.
type Pipelines map[string]*Pipeline

// Extension is the configuration of a service extension. Specific extensions
// must implement this interface and will typically embed ExtensionSettings
// struct or a struct that extends it.
type Extension interface {
	NamedEntity
}

// Extensions is a map of names to extensions.
type Extensions map[string]Extension

// Service defines the configurable components of the service.
type Service struct {
	// Extensions is the ordered list of extensions configured for the service.
	Extensions []string `mapstructure:"extensions"`

	// Pipelines is the set of data pipelines configured for the service.
	Pipelines Pipelines `mapstructure:"pipelines"`
}

// Below are common setting structs for Receivers, Exporters and Processors.
// These are helper structs which you can embed when implementing your specific
// receiver/exporter/processor config storage.

// ReceiverSettings defines common settings for a single-protocol receiver configuration.
// Specific receivers can embed this struct and extend it with more fields if needed.
type ReceiverSettings struct {
	EntityName EntityName `mapstructure:"-"`
	// Endpoint configures the endpoint in the format 'address:port' for the receiver.
	// The default value is set by the receiver populating the struct.
	Endpoint string `mapstructure:"endpoint"`
}

// Name gets the receiver name.
func (rs *ReceiverSettings) Name() EntityName {
	return rs.EntityName
}

// SetName sets the receiver name.
func (rs *ReceiverSettings) SetName(name EntityName) {
	rs.EntityName = name
}

// ExporterSettings defines common settings for an exporter configuration.
// Specific exporters can embed this struct and extend it with more fields if needed.
type ExporterSettings struct {
	EntityName EntityName `mapstructure:"-"`
}

var _ Exporter = (*ExporterSettings)(nil)

// Name gets the exporter name.
func (es *ExporterSettings) Name() EntityName {
	return es.EntityName
}

// SetName sets the exporter name.
func (es *ExporterSettings) SetName(name EntityName) {
	es.EntityName = name
}

// ProcessorSettings defines common settings for a processor configuration.
// Specific processors can embed this struct and extend it with more fields if needed.
type ProcessorSettings struct {
	EntityName EntityName `mapstructure:"-"`
}

// Name gets the processor name.
func (proc *ProcessorSettings) Name() EntityName {
	return proc.EntityName
}

// SetName sets the processor name.
func (proc *ProcessorSettings) SetName(name EntityName) {
	proc.EntityName = name
}

var _ Processor = (*ProcessorSettings)(nil)

// ExtensionSettings defines common settings for a service extension configuration.
// Specific extensions can embed this struct and extend it with more fields if needed.
type ExtensionSettings struct {
	EntityName EntityName `mapstructure:"-"`
}

// Name gets the extension name.
func (ext *ExtensionSettings) Name() EntityName {
	return ext.EntityName
}

// SetName sets the extension name.
func (ext *ExtensionSettings) SetName(name EntityName) {
	ext.EntityName = name
}

var _ Extension = (*ExtensionSettings)(nil)
