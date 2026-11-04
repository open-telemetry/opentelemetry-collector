// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/internal/schemagen"

import (
	"slices"

	"go.yaml.in/yaml/v3"
)

// OrderedPropNames returns md's property names in the order they were declared in
// metadata.yaml.
func OrderedPropNames(md *ConfigMetadata) []string {
	if md == nil {
		return nil
	}
	names := make([]string, 0, len(md.Properties))
	seen := make(map[string]bool, len(md.Properties))
	for _, name := range md.propOrder {
		if _, ok := md.Properties[name]; ok && !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}
	if len(seen) < len(md.Properties) {
		rest := make([]string, 0, len(md.Properties)-len(seen))
		for name := range md.Properties {
			if !seen[name] {
				rest = append(rest, name)
			}
		}
		slices.Sort(rest)
		names = append(names, rest...)
	}
	return names
}

// ApplyConfigsOrder records each property's declaration order from the raw metadata.yaml
// (or external schema file) bytes onto the already-unmarshalled config and
// exported_configs schemas.
func ApplyConfigsOrder(raw []byte, cm *ConfigsMetadata) error {
	if cm == nil {
		return nil
	}
	root, err := rootMappingNode(raw)
	if err != nil {
		return err
	}
	if root == nil {
		return nil
	}
	if cm.Config != nil {
		applyPropOrder(mappingChild(root, "config"), cm.Config)
	}
	if cm.ExportedConfigs != nil {
		applyNamedDefsOrder(mappingChild(root, "exported_configs"), cm.ExportedConfigs)
	}
	return nil
}

// ApplyNamedDefsOrder records each definition's declaration order from a raw YAML
// document whose top level is itself a map of definition name to schema (as opposed to
// a document with "config"/"exported_configs" keys).
func ApplyNamedDefsOrder(raw []byte, defs map[string]*ConfigMetadata) error {
	root, err := rootMappingNode(raw)
	if err != nil {
		return err
	}
	applyNamedDefsOrder(root, defs)
	return nil
}

func rootMappingNode(raw []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return nil, nil
	}
	return root, nil
}

func mappingChild(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func applyNamedDefsOrder(node *yaml.Node, defs map[string]*ConfigMetadata) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i < len(node.Content); i += 2 {
		name := node.Content[i].Value
		if def, ok := defs[name]; ok {
			applyPropOrder(node.Content[i+1], def)
		}
	}
}

func applyPropOrder(node *yaml.Node, md *ConfigMetadata) {
	if node == nil || md == nil {
		return
	}
	if propsNode := mappingChild(node, "properties"); propsNode != nil && propsNode.Kind == yaml.MappingNode {
		for i := 0; i < len(propsNode.Content); i += 2 {
			name := propsNode.Content[i].Value
			md.propOrder = append(md.propOrder, name)
			if prop, ok := md.Properties[name]; ok {
				applyPropOrder(propsNode.Content[i+1], prop)
			}
		}
	}
	if md.Values != nil {
		applyPropOrder(mappingChild(node, "values"), md.Values)
	}
}
