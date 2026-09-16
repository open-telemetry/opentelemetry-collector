// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

type CollectorSection string

const (
	CollectorSectionReceivers  CollectorSection = "receivers"
	CollectorSectionProcessors CollectorSection = "processors"
	CollectorSectionExporters  CollectorSection = "exporters"
	CollectorSectionConnectors CollectorSection = "connectors"
	CollectorSectionExtensions CollectorSection = "extensions"
	CollectorSectionService    CollectorSection = "service"
)

// pipelineSignals lists the signal names accepted as the first part of a
// pipeline identifier under service.pipelines.
var pipelineSignals = []string{
	pipeline.SignalLogs.String(),
	pipeline.SignalMetrics.String(),
	xpipeline.SignalProfiles.String(),
	pipeline.SignalTraces.String(),
}

type CollectorComponentSchema struct {
	Type           string
	DeprecatedType string
	Schema         *JSONSchema
}

type CollectorSchemaParts struct {
	Receivers  []CollectorComponentSchema
	Processors []CollectorComponentSchema
	Exporters  []CollectorComponentSchema
	Connectors []CollectorComponentSchema
	Extensions []CollectorComponentSchema
}

func CombineCollectorSchema(parts CollectorSchemaParts) (*JSONSchema, error) {
	properties := newCollectorProperties(parts)

	sections := []struct {
		name       CollectorSection
		components []CollectorComponentSchema
	}{
		{name: CollectorSectionReceivers, components: parts.Receivers},
		{name: CollectorSectionProcessors, components: parts.Processors},
		{name: CollectorSectionExporters, components: parts.Exporters},
		{name: CollectorSectionConnectors, components: parts.Connectors},
		{name: CollectorSectionExtensions, components: parts.Extensions},
	}

	for _, section := range sections {
		schema, err := combineCollectorComponentSection(section.name, section.components)
		if err != nil {
			return nil, err
		}
		properties[string(section.name)] = schema
	}

	return &JSONSchema{
		Schema:     schemaVersion,
		Type:       "object",
		Properties: properties,
	}, nil
}

func combineCollectorComponentSection(section CollectorSection, components []CollectorComponentSchema) (*JSONSchema, error) {
	sectionSchema := newCollectorComponentSection()

	for _, component := range components {
		if component.Type == "" {
			return nil, fmt.Errorf("%s component type must not be empty", section)
		}

		if err := addCollectorComponentPattern(sectionSchema, component.Type, component.Schema, false); err != nil {
			return nil, fmt.Errorf("%s component %q: %w", section, component.Type, err)
		}

		if component.DeprecatedType != "" {
			if err := addCollectorComponentPattern(sectionSchema, component.DeprecatedType, component.Schema, true); err != nil {
				return nil, fmt.Errorf("%s component %q deprecated type %q: %w", section, component.Type, component.DeprecatedType, err)
			}
		}
	}

	return sectionSchema, nil
}

func addCollectorComponentPattern(section *JSONSchema, componentType string, schema *JSONSchema, deprecated bool) error {
	pattern := collectorComponentPattern(componentType)
	if _, exists := section.PatternProperties[pattern]; exists {
		return fmt.Errorf("duplicate component identifier %q", componentType)
	}

	patternSchema := cloneOrEmptySchema(schema)
	if schema != nil && !reflect.DeepEqual(*schema, JSONSchema{}) {
		patternSchema = &JSONSchema{
			AnyOf: []*JSONSchema{patternSchema, {Type: "null"}},
		}
	}
	if deprecated {
		patternSchema.Deprecated = true
	}
	section.PatternProperties[pattern] = patternSchema

	return nil
}

func collectorComponentPattern(componentType string) string {
	return "^" + regexp.QuoteMeta(componentType) + "(?:/.+)?$"
}

func newCollectorComponentSection() *JSONSchema {
	return &JSONSchema{
		Type:                 "object",
		PatternProperties:    map[string]*JSONSchema{},
		AdditionalProperties: &JSONSchema{Not: &JSONSchema{}},
	}
}

// newCollectorProperties returns the top-level properties that are not
// component sections. Today that is only service, whose extensions and
// pipelines may only reference installed component types. Connectors are
// valid as both pipeline receivers and exporters. service.telemetry is not
// constrained because no schema exists for it yet.
func newCollectorProperties(parts CollectorSchemaParts) map[string]*JSONSchema {
	connectorTypes := collectorComponentTypes(parts.Connectors)

	pipelineSchema := &JSONSchema{
		Type: "object",
		Properties: map[string]*JSONSchema{
			"receivers":  collectorComponentReferenceList(collectorComponentTypes(parts.Receivers), connectorTypes),
			"processors": collectorComponentReferenceList(collectorComponentTypes(parts.Processors)),
			"exporters":  collectorComponentReferenceList(collectorComponentTypes(parts.Exporters), connectorTypes),
		},
		Required:             []string{"receivers", "exporters"},
		AdditionalProperties: &JSONSchema{Not: &JSONSchema{}},
	}
	pipelineSchema.Properties["receivers"].MinItems = new(1)
	pipelineSchema.Properties["exporters"].MinItems = new(1)
	// The collector only rejects repeated identifiers in the processors list.
	// Repeated receivers, exporters and extensions are tolerated at runtime
	// (see open-telemetry/opentelemetry-collector#13912), so the schema must
	// not be stricter than the collector there.
	pipelineSchema.Properties["processors"].UniqueItems = true

	return map[string]*JSONSchema{
		string(CollectorSectionService): {
			Type: "object",
			Properties: map[string]*JSONSchema{
				"extensions": collectorComponentReferenceList(collectorComponentTypes(parts.Extensions)),
				"pipelines": {
					Type: "object",
					PatternProperties: map[string]*JSONSchema{
						collectorIdentifierPattern(pipelineSignals): pipelineSchema,
					},
					AdditionalProperties: &JSONSchema{Not: &JSONSchema{}},
				},
				"telemetry": {
					Description: "Collector internal telemetry settings. Not validated by this schema.",
				},
			},
			AdditionalProperties: &JSONSchema{Not: &JSONSchema{}},
		},
	}
}

// collectorComponentReferenceList returns the schema for an array of component
// identifiers whose type must be one of the given type sets. With no types at
// all the array cannot hold any item, which mirrors a distribution that has no
// component of that role to reference.
func collectorComponentReferenceList(typeSets ...[]string) *JSONSchema {
	var types []string
	for _, set := range typeSets {
		types = append(types, set...)
	}
	slices.Sort(types)
	types = slices.Compact(types)

	items := &JSONSchema{Not: &JSONSchema{}}
	if len(types) > 0 {
		items = &JSONSchema{
			Type:    "string",
			Pattern: collectorIdentifierPattern(types),
		}
	}

	return &JSONSchema{
		Type:  "array",
		Items: items,
	}
}

// collectorComponentTypes returns every type, including deprecated aliases,
// under which the given components can be referenced.
func collectorComponentTypes(components []CollectorComponentSchema) []string {
	types := make([]string, 0, len(components))
	for _, component := range components {
		types = append(types, component.Type)
		if component.DeprecatedType != "" {
			types = append(types, component.DeprecatedType)
		}
	}
	return types
}

// collectorIdentifierPattern matches an identifier of the form type[/name]
// where type is one of the given alternatives.
func collectorIdentifierPattern(types []string) string {
	quoted := make([]string, 0, len(types))
	for _, t := range types {
		quoted = append(quoted, regexp.QuoteMeta(t))
	}
	return "^(?:" + strings.Join(quoted, "|") + ")(?:/.+)?$"
}

func cloneOrEmptySchema(schema *JSONSchema) *JSONSchema {
	if schema == nil {
		return &JSONSchema{}
	}

	cloned := *schema
	return &cloned
}
