// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// PropDoc holds documentation information for a single config property.
type PropDoc struct {
	Name        string
	Type        string
	Default     string
	Required    bool
	Description string
	Deprecated  bool
}

// ExtractPropDocs returns a sorted slice of PropDoc entries for all direct
// properties of cfg. allOf embedded schemas are not expanded; callers that
// want their descriptions should inspect cfg.AllOf separately.
func ExtractPropDocs(cfg *ConfigMetadata) []PropDoc {
	if cfg == nil {
		return nil
	}
	docs := make([]PropDoc, 0, len(cfg.Properties))
	for _, propName := range slices.Sorted(maps.Keys(cfg.Properties)) {
		prop := cfg.Properties[propName]
		docs = append(docs, PropDoc{
			Name:        propName,
			Type:        DocType(prop),
			Default:     DocDefault(prop),
			Required:    slices.Contains(cfg.Required, propName),
			Description: prop.Description,
			Deprecated:  prop.Deprecated,
		})
	}
	return docs
}

// DocType returns a human-readable type label for a ConfigMetadata property,
// suitable for display in generated documentation tables.
func DocType(md *ConfigMetadata) string {
	if md == nil {
		return "any"
	}
	switch md.Type {
	case "string":
		switch md.Format {
		case "duration":
			return "duration"
		case "date-time":
			return "datetime"
		default:
			if len(md.Enum) > 0 {
				return "string (one of: " + strings.Join(enumStrings(md.Enum), ", ") + ")"
			}
			return "string"
		}
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "array":
		if md.Items != nil {
			return "[]" + DocType(md.Items)
		}
		return "[]any"
	case "object":
		if md.AdditionalProperties != nil {
			return "map[string]" + DocType(md.AdditionalProperties)
		}
		return "object"
	default:
		if md.Ref != "" {
			return "object"
		}
		return "any"
	}
}

// DocDefault returns a human-readable representation of a property's default
// value, or an empty string when no default is set.
func DocDefault(md *ConfigMetadata) string {
	if md == nil || md.Default == nil {
		return ""
	}
	return fmt.Sprintf("%v", md.Default)
}

func enumStrings(enum []any) []string {
	s := make([]string, 0, len(enum))
	for _, v := range enum {
		s = append(s, fmt.Sprintf("%v", v))
	}
	return s
}
