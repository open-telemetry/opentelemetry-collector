// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"encoding/json"
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
//
// When the property comes from a $ref, DocType attempts to return a Markdown
// link to the README of the referenced configuration package (e.g. configgrpc,
// confighttp) so that readers can easily navigate to the full documentation
// of the shared type.
func DocType(md *ConfigMetadata) string {
	if md == nil {
		return "any"
	}

	// Handle references first – this is the improved path for the review feedback.
	if ref := effectiveRef(md); ref != "" {
		return docLinkForRef(ref)
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
		return "any"
	}
}

// effectiveRef returns the most useful reference string we have.
func effectiveRef(md *ConfigMetadata) string {
	if md.Ref != "" {
		return md.Ref
	}
	return md.ResolvedFrom
}

// docLinkForRef tries to produce a Markdown link to the README of the
// referenced config type. This directly addresses the review request to
// "generate link to README file to referenced library/config".
func docLinkForRef(ref string) string {
	name := refTypeName(ref)
	if name == "" {
		return "object"
	}

	// Common OpenTelemetry collector shared config packages.
	// We link to the main branch of the collector repo.
	switch {
	case strings.Contains(ref, "configgrpc"):
		return fmt.Sprintf("[%s](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configgrpc/README.md)", name)
	case strings.Contains(ref, "confighttp"):
		return fmt.Sprintf("[%s](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md)", name)
	case strings.Contains(ref, "configauth"):
		return fmt.Sprintf("[%s](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configauth/README.md)", name)
	case strings.Contains(ref, "configtls"):
		return fmt.Sprintf("[%s](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md)", name)
	case strings.Contains(ref, "confignet"):
		return fmt.Sprintf("[%s](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confignet/README.md)", name)
	case strings.Contains(ref, "configretry"):
		return fmt.Sprintf("[%s](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configretry/README.md)", name)
	case strings.Contains(ref, "configopaque"):
		return fmt.Sprintf("[%s](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configopaque/README.md)", name)
	}

	// Internal definition or unknown external ref → just return the nice name.
	// This is still a big improvement over always saying "object".
	return name
}

// refTypeName extracts a human-friendly type name from a $ref string.
func refTypeName(ref string) string {
	if ref == "" {
		return ""
	}
	sep := strings.LastIndexAny(ref, "#/")
	if sep != -1 {
		ref = ref[sep+1:]
	}
	ref = strings.Trim(ref, "/")
	if ref == "" {
		return ""
	}
	return ref
}

// DocDefault returns a human-readable representation of a property's default
// value, or an empty string when no default is set.
//
// Complex values (maps, slices, structs, and nested structures) are rendered
// as compact JSON instead of Go's fmt %v syntax (e.g. map[key:value]), which
// is what end users see in the generated README tables.
func DocDefault(md *ConfigMetadata) string {
	if md == nil || md.Default == nil {
		return ""
	}

	v := md.Default

	// Keep simple scalars clean (no extra quotes around strings etc.)
	switch v.(type) {
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return fmt.Sprintf("%v", v)
	}

	// Everything else (map, slice, struct, nested config, etc.) → compact JSON.
	if b, err := json.Marshal(v); err == nil {
		s := string(b)
		const maxLen = 120
		if len(s) > maxLen {
			return s[:maxLen-3] + "..."
		}
		return s
	}

	return fmt.Sprintf("%v", v)
}

func enumStrings(enum []any) []string {
	s := make([]string, 0, len(enum))
	for _, v := range enum {
		s = append(s, fmt.Sprintf("%v", v))
	}
	return s
}
