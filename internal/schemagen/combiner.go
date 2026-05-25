// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"fmt"
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
	Schema         *ConfigMetadata
}

type CollectorSchemaParts struct {
	Receivers  []CollectorComponentSchema
	Processors []CollectorComponentSchema
	Exporters  []CollectorComponentSchema
	Connectors []CollectorComponentSchema
	Extensions []CollectorComponentSchema
	Service    *ConfigMetadata
}

func CombineCollectorSchema(parts CollectorSchemaParts) (*ConfigMetadata, error) {
	properties := map[string]*ConfigMetadata{
		string(CollectorSectionReceivers):  newCollectorComponentSection(),
		string(CollectorSectionProcessors): newCollectorComponentSection(),
		string(CollectorSectionExporters):  newCollectorComponentSection(),
		string(CollectorSectionConnectors): newCollectorComponentSection(),
		string(CollectorSectionExtensions): newCollectorComponentSection(),
		string(CollectorSectionService):    cloneOrEmptySchema(parts.Service),
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

	return &ConfigMetadata{
		Schema:                      schemaVersion,
		Type:                        "object",
		Properties:                  properties,
		AdditionalPropertiesAllowed: boolPtr(false),
	}, nil
}

func combineCollectorComponentSection(section CollectorSection, components []CollectorComponentSchema) (*ConfigMetadata, error) {
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

func addCollectorComponentPattern(section *ConfigMetadata, componentType string, schema *ConfigMetadata, deprecated bool) error {
	pattern := collectorComponentPattern(componentType)
	if _, exists := section.PatternProperties[pattern]; exists {
		return fmt.Errorf("duplicate component identifier %q", componentType)
	}

	patternSchema := cloneOrEmptySchema(schema)
	if deprecated {
		patternSchema.Deprecated = true
	}
	section.PatternProperties[pattern] = patternSchema

	return nil
}

func collectorComponentPattern(componentType string) string {
	return "^" + regexp.QuoteMeta(componentType) + "(?:/.+)?$"
}

func newCollectorComponentSection() *ConfigMetadata {
	return &ConfigMetadata{
		Type:                        "object",
		PatternProperties:           map[string]*ConfigMetadata{},
		AdditionalPropertiesAllowed: boolPtr(false),
	}
}

func cloneOrEmptySchema(schema *ConfigMetadata) *ConfigMetadata {
	if schema == nil {
		return &ConfigMetadata{}
	}

	cloned := *schema
	return &cloned
}

func boolPtr(v bool) *bool {
	return &v
}
