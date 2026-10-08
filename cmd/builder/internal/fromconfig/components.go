// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package fromconfig resolves the components referenced by a Collector
// configuration to Go modules, in order to generate a builder manifest.
package fromconfig // import "go.opentelemetry.io/collector/cmd/builder/internal/fromconfig"

import (
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Class is a kind of Collector component.
type Class string

const (
	Receiver  Class = "receiver"
	Processor Class = "processor"
	Exporter  Class = "exporter"
	Extension Class = "extension"
	Connector Class = "connector"
)

// Classes lists every component class, in the order sections are reported.
var Classes = []Class{Receiver, Processor, Exporter, Extension, Connector}

// section returns the top-level configuration key holding components of the class.
func (c Class) section() string {
	return string(c) + "s"
}

// Component identifies a component by its class and type.
type Component struct {
	Class Class
	Type  string
}

// ParseComponents reads the receivers, processors, exporters, extensions and
// connectors sections of a Collector configuration and returns the distinct
// components it references, sorted by class then type.
//
// No environment variable expansion is performed.
func ParseComponents(data []byte) ([]Component, error) {
	var cfg map[string]yaml.Node
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	var components []Component
	for _, class := range Classes {
		node, ok := cfg[class.section()]
		if !ok || isNull(&node) {
			continue
		}
		if node.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("section %q must be a mapping to resolve its components", class.section())
		}

		var types []string
		// Mapping node content alternates keys and values.
		for i := 0; i+1 < len(node.Content); i += 2 {
			t := componentType(node.Content[i].Value)
			if !slices.Contains(types, t) {
				types = append(types, t)
			}
		}
		slices.Sort(types)
		for _, t := range types {
			components = append(components, Component{Class: class, Type: t})
		}
	}
	return components, nil
}

func isNull(n *yaml.Node) bool {
	return n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null")
}

// componentType extracts the type from a component ID of the form `type` or `type/name`.
func componentType(id string) string {
	t, _, _ := strings.Cut(id, "/")
	return t
}
