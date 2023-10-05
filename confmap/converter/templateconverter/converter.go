// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package templateconverter // import "go.opentelemetry.io/collector/confmap/converter/templateconverter"

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"go.opentelemetry.io/collector/confmap"
)

// Define a template type by adding a "templates" section to the config.
//
// templates:
//
//	receivers:
//	  my_template: |
//	    receivers: ... # required, one or more
//	    processors: ... # optional
//	    pipelines: ...  # required if processors are used, otherwise optional
//
// Instantiate a templated receiver:
//
// receivers:
//
//	template/my_template[/name]:
//	  parameter_one: value_one
//	  parameter_two: value_two
type converter struct {
	cfg               map[string]any
	receiverTemplates map[string]*template.Template
}

// New returns a confmap.Converter, that renders all templates and inserts them into the given confmap.Conf.
func New() confmap.Converter {
	return &converter{
		receiverTemplates: make(map[string]*template.Template),
	}
}

func (c *converter) Convert(_ context.Context, conf *confmap.Conf) error {
	if err := c.parseTemplates(conf); err != nil {
		return err
	} else if len(c.receiverTemplates) == 0 {
		return nil
	}

	c.cfg = conf.ToStringMap()
	if c.cfg["receivers"] == nil {
		return nil // invalid, but let the unmarshaler handle it
	}
	receiverCfgs, ok := c.cfg["receivers"].(map[string]any)
	if !ok {
		return nil // invalid, but let the unmarshaler handle it
	}
	for templateID, parameters := range receiverCfgs {
		if !strings.HasPrefix(templateID, "template") {
			continue
		}

		id, err := newInstanceID(templateID)
		if err != nil {
			return err
		}

		tmpl, ok := c.receiverTemplates[id.tmplType]
		if !ok {
			return fmt.Errorf("template type %q not found", id.tmplType)
		}

		cfg, err := newTemplateConfig(id, tmpl, parameters)
		if err != nil {
			return err
		}

		c.expandTemplate(cfg, id)
	}
	delete(c.cfg, "templates")

	*conf = *confmap.NewFromStringMap(c.cfg)
	return nil
}

func (c *converter) parseTemplates(conf *confmap.Conf) error {
	if !conf.IsSet("templates") {
		return nil
	}

	templatesMap, ok := conf.ToStringMap()["templates"].(map[string]any)
	if !ok {
		return fmt.Errorf("'templates' must be a map")
	}
	if templatesMap["receivers"] == nil {
		return fmt.Errorf("'templates' must contain a 'receivers' section")
	}

	receiverTemplates, ok := templatesMap["receivers"].(map[string]any)
	if !ok {
		return fmt.Errorf("'templates::receivers' must be a map")
	}

	for templateType, templateVal := range receiverTemplates {
		templateStr, ok := templateVal.(string)
		if !ok {
			return fmt.Errorf("'templates::receivers::%s' must be a string", templateType)
		}
		parsedTemplate, err := template.New(templateType).Parse(templateStr)
		if err != nil {
			return err
		}
		c.receiverTemplates[templateType] = parsedTemplate
	}
	return nil
}

func (c *converter) expandTemplate(cfg *templateConfig, id *instanceID) {
	// Delete the reference to this template instance and
	// replace it with the rendered receiver(s).
	delete(c.cfg["receivers"].(map[string]any), id.withPrefix("template"))
	for receiverID, receiverCfg := range cfg.Receivers {
		c.cfg["receivers"].(map[string]any)[receiverID] = receiverCfg
	}

	// Special case where the template only contains receivers. In this case,
	// we can just substitute the rendered receivers in place of the template ID.
	if len(cfg.Processors) == 0 && len(cfg.Pipelines) == 0 {
		c.expandSimple(cfg, id)
	} else {
		c.expandComplex(cfg, id)
	}
}

// Special case where the template only contains receivers. In this case,
// we can just substitute the rendered receivers in place of the template ID.
func (c *converter) expandSimple(cfg *templateConfig, id *instanceID) {
	pipelinesMap := c.cfg["service"].(map[string]any)["pipelines"].(map[string]any)
	for _, pipelineMap := range pipelinesMap {
		receiverIDs := pipelineMap.(map[string]any)["receivers"].([]any)
		newReceiverIDs := make([]any, 0, len(receiverIDs))
		for _, receiverID := range receiverIDs {
			if receiverID != id.withPrefix("template") {
				newReceiverIDs = append(newReceiverIDs, receiverID)
			}
		}

		if len(newReceiverIDs) == len(receiverIDs) {
			// This pipeline did not use the template
			continue
		}

		for receiverID := range cfg.Receivers {
			newReceiverIDs = append(newReceiverIDs, receiverID)
		}
		// This makes tests deterministic
		sort.Slice(newReceiverIDs, func(i, j int) bool {
			return newReceiverIDs[i].(string) < newReceiverIDs[j].(string)
		})
		pipelineMap.(map[string]any)["receivers"] = newReceiverIDs
	}
}

// Any partial pipelines defined in a template will be expanded into full pipelines.
// These will all use a forward connector to emit data to the pipelines in which
// the template was used as a receiver.
func (c *converter) expandComplex(cfg *templateConfig, id *instanceID) {
	pipelinesMap := c.cfg["service"].(map[string]any)["pipelines"].(map[string]any)
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
	connectorID := id.withPrefix("forward")
	if c.cfg["connectors"] == nil {
		c.cfg["connectors"] = make(map[string]any, 1)
	}
	c.cfg["connectors"].(map[string]any)[connectorID] = nil

	// Crawl through existing "pipelines" and replace all references
	// to this instance of the template with the forward connector.
	//
	// Also take note of the pipeline data types in which the template is used.
	// We'll use these later to include only relevant partial pipelines.
	var usedTraces, usedMetrics, usedLogs bool
	for pipelineID := range pipelinesMap {
		switch {
		case isTraces(pipelineID):
			usedTraces = true
		case isMetrics(pipelineID):
			usedMetrics = true
		case isLogs(pipelineID):
			usedLogs = true
		default:
			continue
		}

		pipelineMap := pipelinesMap[pipelineID].(map[string]any)
		receiverIDs := pipelineMap["receivers"].([]any)
		for i, receiverID := range receiverIDs {
			if receiverID == id.withPrefix("template") {
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
			continue
		}

		scopedPipelineName := id.withPrefix(partialName)
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
