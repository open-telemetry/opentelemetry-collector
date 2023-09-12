// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package templateconverter // import "go.opentelemetry.io/collector/confmap/converter/templateconverter"

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/confmap"
)

// Using a template requires the following format
// receivers:
//
//	template/template_type[/name]:
//	  parameter_one: value_one
//	  parameter_two: value_two
type converter struct {
	cfg map[string]any
}

// New returns a confmap.Converter, that renders all templates and inserts them into the given confmap.Conf.
//
// Notice: This API is experimental.
func New() confmap.Converter {
	return &converter{}
}

func (c *converter) Convert(_ context.Context, conf *confmap.Conf) error {
	if !conf.IsSet("templates") {
		return nil
	}
	c.cfg = conf.ToStringMap()

	for templateID := range c.cfg["receivers"].(map[string]any) {
		if !strings.HasPrefix(templateID, "template/") {
			continue
		}
		var cfg *TemplateConfig
		cfg, err := c.renderTemplate(templateID)
		if err != nil {
			return fmt.Errorf("template %q: %w", templateID, err)
		}
		err = c.expandTemplate(cfg, templateID)
		if err != nil {
			return err
		}
	}
	delete(c.cfg, "templates")

	*conf = *confmap.NewFromStringMap(c.cfg)
	return nil
}

func (c *converter) renderTemplate(templateID string) (*TemplateConfig, error) {
	templateType, instanceName, err := parseTemplateID(templateID)
	if err != nil {
		return nil, err
	}
	template, ok := c.cfg["templates"].(map[string]any)[templateType]
	if !ok {
		return nil, fmt.Errorf("template type %q not found", templateType)
	}
	return render(templateType, instanceName, template.(string), c.cfg["receivers"].(map[string]any)[templateID])
}

func render(templateType, instanceName, rawTemplate string, parameters any) (*TemplateConfig, error) {
	var parsedTemplate *template.Template
	parsedTemplate, err := template.New(templateType).Parse(rawTemplate)
	if err != nil {
		return nil, err
	}

	var rendered bytes.Buffer
	if err = parsedTemplate.Execute(&rendered, parameters); err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}

	cfg := new(TemplateConfig)
	if err = yaml.Unmarshal(rendered.Bytes(), cfg); err != nil {
		return nil, fmt.Errorf("malformed: %w", err)
	}

	if len(cfg.Receivers) == 0 {
		return nil, errors.New("must have at least one receiver")
	}

	if len(cfg.Pipelines) == 0 {
		return nil, errors.New("must have at least one pipeline")
	}

	// Apply a scope to all component IDs in the template.
	// At a minimum, the this appends the template type, but will
	// also apply the template instance name if possible.
	scopeMapKeys(cfg.Receivers, templateType, instanceName)
	scopeMapKeys(cfg.Processors, templateType, instanceName)
	for _, p := range cfg.Pipelines {
		scopeSliceValues(p.Receivers, templateType, instanceName)
		scopeSliceValues(p.Processors, templateType, instanceName)
	}

	return cfg, nil
}

func (c *converter) expandTemplate(cfg *TemplateConfig, templateID string) error {
	templateType, instanceName, err := parseTemplateID(templateID)
	if err != nil {
		return err
	}

	// Update the receivers section by deleting the reference to the
	// template and replacing it with the rendered receiver(s).
	delete(c.cfg["receivers"].(map[string]any), templateID)
	for receiverID, receiverCfg := range cfg.Receivers {
		c.cfg["receivers"].(map[string]any)[receiverID] = receiverCfg
	}

	// Update the processors section by adding any rendered processors.
	if len(cfg.Processors) > 0 && c.cfg["processors"] == nil {
		c.cfg["processors"] = make(map[string]any, len(cfg.Processors))
	}
	for processorID, processorCfg := range cfg.Processors {
		c.cfg["processors"].(map[string]any)[processorID] = processorCfg
	}

	// Add a dedicated forward connector for this template instance.
	// This will consume data from the partial pipelines and emit to
	// any top-level pipelines that used the template as a receiver.
	connectorID := "forward/" + templateType
	if instanceName != "" {
		connectorID = connectorID + "/" + instanceName
	}
	if c.cfg["connectors"] == nil {
		c.cfg["connectors"] = make(map[string]any, 1)
	}
	c.cfg["connectors"].(map[string]any)[connectorID] = nil

	// Crawl through existing "pipelines" and replace all references
	// to this instance of the template with the forward connector.
	// Since these are a 1:1 changes, we are updating the original Conf.
	//
	// Also take note of the pipeline data types in which the template is used.
	// We'll use these later to include only relevant partial pipelines.
	var usedTraces, usedMetrics, usedLogs bool
	pipelinesMap := c.cfg["service"].(map[string]any)["pipelines"].(map[string]any)
	for pipelineID := range pipelinesMap {
		switch {
		case isTraces(pipelineID):
			usedTraces = true
		case isMetrics(pipelineID):
			usedMetrics = true
		case isLogs(pipelineID):
			usedLogs = true
		default:
			// For now, just let it be. It'll blow up later in config
			// validation but that's not the converter's problem to flag.
			continue
		}

		pipelineMap := pipelinesMap[pipelineID].(map[string]any)
		receiverIDs := pipelineMap["receivers"].([]any)
		for i, receiverID := range receiverIDs {
			if receiverID == templateID {
				receiverIDs[i] = connectorID
			}
		}
		pipelineMap["receivers"] = receiverIDs
	}

	// For each partial pipeline, build a full pipeline by appending
	// the forward connector as the exporter.
	//
	// Only include the pipeline if the template was used as a receiver
	// in a pipeline of the same data type.
	for partialName, partial := range cfg.Pipelines {
		switch {
		case isTraces(partialName):
			if !usedTraces {
				continue
			}
		case isMetrics(partialName):
			if !usedMetrics {
				continue
			}
		case isLogs(partialName):
			if !usedLogs {
				continue
			}
		default:
			return fmt.Errorf("pipeline id must start with data type: %q", partialName)
		}

		scopedPipelineName := partialName + "/" + templateType
		if instanceName != "" {
			scopedPipelineName += "/" + instanceName
		}
		receivers := make([]any, 0, len(partial.Receivers))
		for _, receiverID := range partial.Receivers {
			receivers = append(receivers, receiverID)
		}
		processors := make([]any, 0, len(partial.Processors))
		for _, processorID := range partial.Processors {
			processors = append(processors, processorID)
		}
		newPipeline := map[string]any{
			"receivers": receivers,
			"exporters": []any{connectorID},
		}
		if len(partial.Processors) > 0 {
			newPipeline["processors"] = processors
		}
		pipelinesMap[scopedPipelineName] = newPipeline
	}
	return nil
}

type TemplateConfig struct {
	Receivers  map[string]any             `yaml:"receivers,omitempty"`
	Processors map[string]any             `yaml:"processors,omitempty"`
	Pipelines  map[string]PartialPipeline `yaml:"pipelines,omitempty"`
}

type PartialPipeline struct {
	Receivers  []string `yaml:"receivers,omitempty"`
	Processors []string `yaml:"processors,omitempty"`
}

func parseTemplateID(templateID string) (templateType, instanceName string, err error) {
	templateIDParts := strings.SplitN(templateID, "/", 3)
	switch len(templateIDParts) {
	case 1: // just "template"
		return "", "", fmt.Errorf("'template' must be followed by type")
	case 2: // template/type
		return templateIDParts[1], "", nil
	case 3: // template/type/name
		return templateIDParts[1], templateIDParts[2], nil
	default:
		return "", "", fmt.Errorf("invalid template ID: %q", templateID)
	}
}

// In order to ensure the component IDs in this template are globally unique,
// appending the template instance name to the ID of each component.
func scopeMapKeys(cfgs map[string]any, templateType, instanceName string) {
	// To avoid risk of collision, build a list of IDs,
	// sort them by length, and then append starting with the longest.
	ids := make([]string, 0, len(cfgs))
	for id := range cfgs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return len(ids[i]) > len(ids[j])
	})
	for _, id := range ids {
		cfg := cfgs[id]
		delete(cfgs, id)
		cfgs[scopedID(id, templateType, instanceName)] = cfg
	}
}

func scopeSliceValues(ids []string, templateType, instanceName string) {
	for i, id := range ids {
		ids[i] = scopedID(id, templateType, instanceName)
	}
}

func scopedID(id string, templateType, instanceName string) string {
	if instanceName == "" {
		return id + "/" + templateType
	}
	return id + "/" + templateType + "/" + instanceName
}

func isTraces(pipelineID string) bool {
	return pipelineID == "traces" || strings.HasPrefix(pipelineID, "traces/")
}

func isMetrics(pipelineID string) bool {
	return pipelineID == "metrics" || strings.HasPrefix(pipelineID, "metrics/")
}

func isLogs(pipelineID string) bool {
	return pipelineID == "logs" || strings.HasPrefix(pipelineID, "logs/")
}
