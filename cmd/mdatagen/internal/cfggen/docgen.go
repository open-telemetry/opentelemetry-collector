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
	// Scalar is true for non-object definitions (type aliases such as an enum
	// string or a bounded int), which render as a single summary row.
	Scalar bool
}

// CfgRootSection wraps a schema for the top-level configSection call.
func CfgRootSection(cfg *ConfigMetadata) DocSection {
	return DocSection{Schema: cfg}
}

// CfgSubSection wraps a schema for a recursive configSection call.
func CfgSubSection(cfg *ConfigMetadata, anchor, title string) DocSection {
	return DocSection{Schema: cfg, Anchor: anchor, Title: title}
}

// CfgPropDocs returns a slice of PropDoc, in declaration order, for all YAML-visible
// properties of cfg: direct Properties plus properties flattened from embeds.
func CfgPropDocs(cfg *ConfigMetadata) []PropDoc {
	if cfg == nil {
		return nil
	}
	var docs []PropDoc
	collectPropDocs(cfg, &docs)
	return docs
}

func collectPropDocs(cfg *ConfigMetadata, docs *[]PropDoc) {
	for _, name := range OrderedPropNames(cfg) {
		prop := cfg.Properties[name]

		if prop.Embed {
			collectPropDocs(prop, docs)
			continue
		}

		*docs = append(*docs, PropDoc{
			Name:        name,
			Schema:      prop,
			Required:    slices.Contains(cfg.Required, name),
			Description: prop.Description,
			Deprecated:  prop.Deprecated != nil,
		})
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
	case StringType:
		if cfg.GoStruct.Type == "time.Duration" || cfg.Format == "duration" {
			return "duration"
		}
		if cfg.GoStruct.Type == "time.Time" || cfg.Format == "date-time" {
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
	case SliceType:
		if cfg.Values != nil {
			return "[]" + CfgDocType(cfg.Values)
		}
		return "[]any"
	case MapType:
		if cfg.Values != nil {
			return "map[string]" + CfgDocType(cfg.Values)
		}
		return "map[string]any"
	default:
		return string(cfg.Type)
	}
}

// CfgDocConstraints returns a compact, human-readable summary of the validation
// constraints on a scalar schema (numeric bounds, length bounds, pattern).
// It returns an empty string when there are none.
func CfgDocConstraints(cfg *ConfigMetadata) string {
	if cfg == nil {
		return ""
	}
	var parts []string
	if cfg.Minimum != nil {
		parts = append(parts, fmt.Sprintf("min: %v", *cfg.Minimum))
	}
	if cfg.ExclusiveMinimum != nil {
		parts = append(parts, fmt.Sprintf("exclusive min: %v", *cfg.ExclusiveMinimum))
	}
	if cfg.Maximum != nil {
		parts = append(parts, fmt.Sprintf("max: %v", *cfg.Maximum))
	}
	if cfg.ExclusiveMaximum != nil {
		parts = append(parts, fmt.Sprintf("exclusive max: %v", *cfg.ExclusiveMaximum))
	}
	if cfg.MinLength != nil {
		parts = append(parts, fmt.Sprintf("min length: %d", *cfg.MinLength))
	}
	if cfg.MaxLength != nil {
		parts = append(parts, fmt.Sprintf("max length: %d", *cfg.MaxLength))
	}
	if cfg.Pattern != "" {
		parts = append(parts, fmt.Sprintf("pattern: %#q", cfg.Pattern))
	}
	return strings.Join(parts, ", ")
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
		"cfgPropDocs":         CfgPropDocs,
		"cfgIsObject":         CfgIsObject,
		"cfgAnchor":           CfgAnchor,
		"cfgDocType":          CfgDocType,
		"cfgDocDefault":       CfgDocDefault,
		"cfgRootSection":      CfgRootSection,
		"cfgSubSection":       CfgSubSection,
		"cfgExportedSections": CfgExportedSections,
		"cfgDocConstraints":   CfgDocConstraints,
	}
}

// CfgExportedSections returns one DocSection for every user-facing definition
// under exported_configs, including scalar type aliases. InternalOnly
// definitions (injected by mdatagen itself) are skipped. Sections are sorted
// by name so the generated README is stable.
func CfgExportedSections(md *ConfigsMetadata) []DocSection {
	if md == nil {
		return nil
	}
	var sections []DocSection
	for _, name := range slices.Sorted(maps.Keys(md.ExportedConfigs)) {
		cfg := md.ExportedConfigs[name]
		if cfg == nil || cfg.InternalOnly {
			continue
		}
		anchor := CfgAnchor("exported_configs", name)
		sec := CfgSubSection(cfg, anchor, anchor)
		sec.Scalar = !CfgIsObject(cfg)
		sections = append(sections, sec)
	}
	return sections
}
