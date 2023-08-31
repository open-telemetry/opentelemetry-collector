// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package templateconverter // import "go.opentelemetry.io/collector/confmap/converter/templateconverter"

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type instanceID struct {
	tmplType string
	instName string
}

func newInstanceID(id string) (*instanceID, error) {
	parts := strings.SplitN(id, "/", 3)
	switch len(parts) {
	case 2: // template/type
		return &instanceID{
			tmplType: parts[1],
		}, nil
	case 3: // template/type/name
		return &instanceID{
			tmplType: parts[1],
			instName: parts[2],
		}, nil
	}
	return nil, fmt.Errorf("'template' must be followed by type")
}

func (id *instanceID) withPrefix(prefix string) string {
	if id.instName == "" {
		return prefix + "/" + id.tmplType
	}
	return prefix + "/" + id.tmplType + "/" + id.instName
}

type templateConfig struct {
	Receivers  map[string]any             `yaml:"receivers,omitempty"`
	Processors map[string]any             `yaml:"processors,omitempty"`
	Pipelines  map[string]partialPipeline `yaml:"pipelines,omitempty"`
}

type partialPipeline struct {
	Receivers  []string `yaml:"receivers,omitempty"`
	Processors []string `yaml:"processors,omitempty"`
}

func newTemplateConfig(id *instanceID, tmpl *template.Template, parameters any) (*templateConfig, error) {
	var rendered bytes.Buffer
	var err error
	if err = tmpl.Execute(&rendered, parameters); err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}

	cfg := new(templateConfig)
	if err = yaml.Unmarshal(rendered.Bytes(), cfg); err != nil {
		return nil, fmt.Errorf("malformed: %w", err)
	}

	if len(cfg.Receivers) == 0 {
		return nil, errors.New("must have at least one receiver")
	}

	if len(cfg.Processors) > 0 && len(cfg.Pipelines) == 0 {
		return nil, errors.New("template containing processors must have at least one pipeline")
	}
	cfg.applyScope(id.tmplType, id.instName)
	return cfg, nil
}

func (cfg *templateConfig) applyScope(tmplType, instName string) {
	// Apply a scope to all component IDs in the template.
	// At a minimum, the this appends the template type, but will
	// also apply the template instance name if possible.
	scopeMapKeys(cfg.Receivers, tmplType, instName)
	scopeMapKeys(cfg.Processors, tmplType, instName)
	for _, p := range cfg.Pipelines {
		scopeSliceValues(p.Receivers, tmplType, instName)
		scopeSliceValues(p.Processors, tmplType, instName)
	}
}

// In order to ensure the component IDs in this template are globally unique,
// appending the template instance name to the ID of each component.
func scopeMapKeys(cfgs map[string]any, tmplType, instName string) {
	// To avoid risk of collision, build a list of IDs,
	// sort them by length, and then append starting with the longest.
	componentIDs := make([]string, 0, len(cfgs))
	for componentID := range cfgs {
		componentIDs = append(componentIDs, componentID)
	}
	sort.Slice(componentIDs, func(i, j int) bool {
		return len(componentIDs[i]) > len(componentIDs[j])
	})
	for _, componentID := range componentIDs {
		cfg := cfgs[componentID]
		delete(cfgs, componentID)
		cfgs[scopedID(componentID, tmplType, instName)] = cfg
	}
}

func scopeSliceValues(componentIDs []string, tmplType, instName string) {
	for i, componentID := range componentIDs {
		componentIDs[i] = scopedID(componentID, tmplType, instName)
	}
}

func scopedID(componentID string, templateType, instanceName string) string {
	if instanceName == "" {
		return componentID + "/" + templateType
	}
	return componentID + "/" + templateType + "/" + instanceName
}
