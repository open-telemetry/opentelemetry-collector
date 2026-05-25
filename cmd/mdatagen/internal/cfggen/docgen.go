// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfggen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfggen"

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// PropDoc holds documentation for a single config property row.
type PropDoc struct {
	Name        string
	Schema      *ConfigMetadata
	Required    bool
	Description string
	Deprecated  bool
}

// DocSection is the data passed to the configSection template block.
type DocSection struct {
	Schema *ConfigMetadata
	Anchor string
	Title  string
}

// CfgRootSection wraps a schema for the top-level configSection call.
func CfgRootSection(cfg *ConfigMetadata) DocSection {
	return DocSection{Schema: cfg}
}

// CfgSubSection wraps a schema for a recursive configSection call.
func CfgSubSection(cfg *ConfigMetadata, anchor, title string) DocSection {
	return DocSection{Schema: cfg, Anchor: anchor, Title: title}
}

// CfgPropDocs returns a sorted slice of PropDoc for all YAML-visible properties
// of cfg: direct Properties plus properties flattened from AllOf embeds.
func CfgPropDocs(cfg *ConfigMetadata) []PropDoc {
	if cfg == nil {
		return nil
	}
	var docs []PropDoc
	collectPropDocs(cfg, &docs)
	slices.SortFunc(docs, func(a, b PropDoc) int {
		return strings.Compare(a.Name, b.Name)
	})
	return docs
}

func collectPropDocs(cfg *ConfigMetadata, docs *[]PropDoc) {
	for _, name := range slices.Sorted(maps.Keys(cfg.Properties)) {
		prop := cfg.Properties[name]
		*docs = append(*docs, PropDoc{
			Name:        name,
			Schema:      prop,
			Required:    slices.Contains(cfg.Required, name),
			Description: prop.Description,
			Deprecated:  prop.Deprecated,
		})
	}
	for _, embed := range cfg.AllOf {
		if embed != nil {
			collectPropDocs(embed, docs)
		}
	}
}

// CfgIsObject returns true if the schema has its own Properties — meaning the
// template should render it as an inline sub-section rather than a scalar type.
func CfgIsObject(cfg *ConfigMetadata) bool {
	return cfg != nil && len(cfg.Properties) > 0
}

// CfgAnchor builds a dotted anchor string for a property within a parent section.
// When parent is empty the anchor is just the name.
func CfgAnchor(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "." + name
}

// CfgDocType returns a human-readable type label for a ConfigMetadata property.
func CfgDocType(cfg *ConfigMetadata) string {
	if cfg == nil {
		return "any"
	}
	if len(cfg.Properties) > 0 {
		return "object"
	}
	switch cfg.Type {
	case "string":
		if cfg.GoType == "time.Duration" || cfg.Format == "duration" {
			return "duration"
		}
		if cfg.GoType == "time.Time" || cfg.Format == "date-time" {
			return "datetime"
		}
		if len(cfg.Enum) > 0 {
			vals := make([]string, 0, len(cfg.Enum))
			for _, v := range cfg.Enum {
				vals = append(vals, fmt.Sprintf("%v", v))
			}
			return "string (one of: " + strings.Join(vals, ", ") + ")"
		}
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "array":
		if cfg.Items != nil {
			return "[]" + CfgDocType(cfg.Items)
		}
		return "[]any"
	case "object":
		if cfg.AdditionalProperties != nil {
			return "map[string]" + CfgDocType(cfg.AdditionalProperties)
		}
		return "object"
	default:
		return "any"
	}
}

// CfgDocDefault returns a human-readable default value for display in the docs table.
// Primitives are returned as-is; slices and maps are marshaled to a compact single-line
// YAML flow representation to avoid breaking Markdown table cells.
func CfgDocDefault(cfg *ConfigMetadata) string {
	if cfg == nil || cfg.Default == nil {
		return ""
	}
	switch v := cfg.Default.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return marshalFlowYAML(v)
	}
}

func marshalFlowYAML(v any) string {
	node := &yaml.Node{}
	if err := node.Encode(v); err != nil {
		return fmt.Sprintf("%v", v)
	}
	setFlowStyle(node)
	out, err := yaml.Marshal(node)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return strings.TrimRight(string(out), "\n")
}

func setFlowStyle(n *yaml.Node) {
	if n == nil {
		return
	}
	switch n.Kind {
	case yaml.SequenceNode, yaml.MappingNode:
		n.Style = yaml.FlowStyle
	}
	for _, child := range n.Content {
		setFlowStyle(child)
	}
}

// NewCfgDocFns returns template functions for config documentation generation.
// These are added to the func map by WithCfgFns.
func NewCfgDocFns() map[string]any {
	return map[string]any{
		"cfgPropDocs":    CfgPropDocs,
		"cfgIsObject":    CfgIsObject,
		"cfgAnchor":      CfgAnchor,
		"cfgDocType":     CfgDocType,
		"cfgDocDefault":  CfgDocDefault,
		"cfgRootSection": CfgRootSection,
		"cfgSubSection":  CfgSubSection,
	}
}
