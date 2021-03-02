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
package configmodels

import (
	"errors"
	"strings"
)

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
	Receivers
	Exporters
	Processors
	Extensions
	Service
}

// typeAndNameSeparator is the separator that is used between type and name in type/name composite keys.
const typeAndNameSeparator = "/"

// Type is the component type as it is used in the config.
type Type string

// NamedEntity is a configuration entity that has a type and a name.
type NamedEntity struct {
	typeVal Type
	nameVal string
}

// NewNamedEntity returns a new NamedEntity with the given type and name.
func NewNamedEntity(typeVal Type, nameVal string) NamedEntity {
	return NamedEntity{typeVal: typeVal, nameVal: nameVal}
}

// ParseNamedEntity parses the NamedEntity from a full name "type/name" format.
func ParseNamedEntity(fullName string) (NamedEntity, error) {
	tVal, nVal, err := DecodeTypeAndName(fullName)
	if err != nil {
		return NamedEntity{}, err
	}
	return NewNamedEntity(Type(tVal), nVal), nil
}

func (nc NamedEntity) Type() Type {
	return nc.typeVal
}

func (nc NamedEntity) Name() string {
	return nc.nameVal
}

func (nc NamedEntity) FullName() string {
	if nc.nameVal == "" {
		return string(nc.typeVal)
	}
	return string(nc.typeVal) + typeAndNameSeparator + nc.nameVal
}

// Receiver is the configuration of a receiver. Specific receivers must implement this
// interface and will typically embed ReceiverSettings struct or a struct that extends it.
type Receiver interface {
}

// Receivers is a map of names to Receivers.
type Receivers map[NamedEntity]Receiver

// Exporter is the configuration of an exporter.
type Exporter interface {
}

// Exporters is a map of names to Exporters.
type Exporters map[NamedEntity]Exporter

// Processor is the configuration of a processor. Specific processors must implement this
// interface and will typically embed ProcessorSettings struct or a struct that extends it.
type Processor interface {
}

// Processors is a map of names to Processors.
type Processors map[NamedEntity]Processor

// Extension is the configuration of a service extension. Specific extensions
// must implement this interface and will typically embed ExtensionSettings
// struct or a struct that extends it.
type Extension interface {
}

// Extensions is a map of names to extensions.
type Extensions map[NamedEntity]Extension

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
	Receivers  []NamedEntity
	Processors []NamedEntity
	Exporters  []NamedEntity
}

// Pipelines is a map of names to Pipelines.
type Pipelines map[string]*Pipeline

// Service defines the configurable components of the service.
type Service struct {
	// Extensions is the ordered list of extensions configured for the service.
	Extensions []NamedEntity

	// Pipelines is the set of data pipelines configured for the service.
	Pipelines Pipelines `mapstructure:"pipelines"`
}

// Below are common setting structs for Receivers, Exporters and Processors.
// These are helper structs which you can embed when implementing your specific
// receiver/exporter/processor config storage.

// ReceiverSettings defines common settings for a receiver configuration.
// Specific receivers can embed this struct and extend it with more fields if needed.
type ReceiverSettings struct {
	TypeVal Type   `mapstructure:"-"`
	NameVal string `mapstructure:"-"`
}

// Name gets the receiver name.
func (rs *ReceiverSettings) Name() string {
	return rs.NameVal
}

// SetName sets the receiver name.
func (rs *ReceiverSettings) SetName(name string) {
	rs.NameVal = name
}

// Type sets the receiver type.
func (rs *ReceiverSettings) Type() Type {
	return rs.TypeVal
}

// ExporterSettings defines common settings for an exporter configuration.
// Specific exporters can embed this struct and extend it with more fields if needed.
type ExporterSettings struct {
	TypeVal Type   `mapstructure:"-"`
	NameVal string `mapstructure:"-"`
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
func (es *ExporterSettings) Type() Type {
	return es.TypeVal
}

// ProcessorSettings defines common settings for a processor configuration.
// Specific processors can embed this struct and extend it with more fields if needed.
type ProcessorSettings struct {
	TypeVal Type   `mapstructure:"-"`
	NameVal string `mapstructure:"-"`
}

// Name gets the processor name.
func (proc *ProcessorSettings) Name() string {
	return proc.NameVal
}

// SetName sets the processor name.
func (proc *ProcessorSettings) SetName(name string) {
	proc.NameVal = name
}

// Type sets the processor type.
func (proc *ProcessorSettings) Type() Type {
	return proc.TypeVal
}

var _ Processor = (*ProcessorSettings)(nil)

// ExtensionSettings defines common settings for a service extension configuration.
// Specific extensions can embed this struct and extend it with more fields if needed.
type ExtensionSettings struct {
	TypeVal Type   `mapstructure:"-"`
	NameVal string `mapstructure:"-"`
}

// Name gets the extension name.
func (ext *ExtensionSettings) Name() string {
	return ext.NameVal
}

// SetName sets the extension name.
func (ext *ExtensionSettings) SetName(name string) {
	ext.NameVal = name
}

// Type sets the extension type.
func (ext *ExtensionSettings) Type() Type {
	return ext.TypeVal
}

var _ Extension = (*ExtensionSettings)(nil)

// DecodeTypeAndName decodes the type and name from a full name "type/name" format.
func DecodeTypeAndName(fullName string) (string, string, error) {
	items := strings.SplitN(fullName, typeAndNameSeparator, 2)
	if len(items) == 0 || items[0] == "" {
		return "", "", errors.New("type/name format must have the type part")
	}

	var nameVal string
	if len(items) > 1 {
		// "name" part is present.
		nameVal = strings.TrimSpace(items[1])
		if nameVal == "" {
			return "", "", errors.New("empty name, consider to remove " + typeAndNameSeparator + " suffix")
		}
	} else {
		nameVal = ""
	}

	return strings.TrimSpace(items[0]), nameVal, nil
}
