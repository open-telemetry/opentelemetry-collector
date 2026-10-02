// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"fmt"
	"reflect"
	"regexp"
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
	properties := map[string]*JSONSchema{
		string(CollectorSectionReceivers):  newCollectorComponentSection(),
		string(CollectorSectionProcessors): newCollectorComponentSection(),
		string(CollectorSectionExporters):  newCollectorComponentSection(),
		string(CollectorSectionConnectors): newCollectorComponentSection(),
		string(CollectorSectionExtensions): newCollectorComponentSection(),
		string(CollectorSectionService):    newCollectorServiceSection(parts),
	}

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

func newCollectorServiceSection(parts CollectorSchemaParts) *JSONSchema {
	return &JSONSchema{
		Type: "object",
		Properties: map[string]*JSONSchema{
			"telemetry": {Type: "object"},
			"extensions": {
				Type:  "array",
				Items: componentReferenceSchema(parts.Extensions),
			},
			"pipelines": {
				Type: "object",
				PatternProperties: map[string]*JSONSchema{
					"^(?:traces|metrics|logs)(?:/.+)?$": {
						Type: "object",
						Properties: map[string]*JSONSchema{
							"receivers": {
								Type:     "array",
								MinItems: func() *int { i := 1; return &i }(),
								Items:    componentReferenceSchema(parts.Receivers),
							},
							"processors": {
								Type:  "array",
								Items: componentReferenceSchema(parts.Processors),
							},
							"exporters": {
								Type:     "array",
								MinItems: func() *int { i := 1; return &i }(),
								Items:    componentReferenceSchema(parts.Exporters),
							},
						},
						Required:             []string{"receivers", "exporters"},
						AdditionalProperties: &JSONSchema{Not: &JSONSchema{}},
					},
				},
				AdditionalProperties: &JSONSchema{Not: &JSONSchema{}},
			},
		},
		AdditionalProperties: &JSONSchema{Not: &JSONSchema{}},
	}
}

func componentReferenceSchema(components []CollectorComponentSchema) *JSONSchema {
	anyOf := make([]*JSONSchema, 0, len(components)*2)

	for _, component := range components {
		if component.Type != "" {
			anyOf = append(anyOf, &JSONSchema{
				Pattern: collectorComponentPattern(component.Type),
			})
		}
		if component.DeprecatedType != "" {
			anyOf = append(anyOf, &JSONSchema{
				Pattern: collectorComponentPattern(component.DeprecatedType),
			})
		}
	}

	return &JSONSchema{
		AnyOf: anyOf,
	}
}

func cloneOrEmptySchema(schema *JSONSchema) *JSONSchema {
	if schema == nil {
		return &JSONSchema{}
	}

	cloned := *schema
	return &cloned
}
