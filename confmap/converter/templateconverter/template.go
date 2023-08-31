// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package templateconverter // import "go.opentelemetry.io/collector/confmap/converter/templateconverter"

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/confmap"
)

type converter struct{}

// New returns a confmap.Converter, that renders all templates and inserts them into the given confmap.Conf.
//
// Notice: This API is experimental.
func New() confmap.Converter {
	return converter{}
}

func (converter) Convert(_ context.Context, conf *confmap.Conf) error {
	if !conf.IsSet("templates") {
		return nil
	}

	receiversConf, err := conf.Sub("receivers")
	if err != nil {
		return err
	}
	// We will make destructive changes to this which cannot be merged
	// back onto the original Conf. We'll maintain this as a raw map
	// until the end and then rebuild the Conf using the updated map.
	receiversMap := receiversConf.ToStringMap()

	processorsConf, err := conf.Sub("processors")
	if err != nil {
		return err
	}

	connectorsConf, err := conf.Sub("connectors")
	if err != nil {
		return err
	}

	serviceConf, err := conf.Sub("service")
	if err != nil {
		return err
	}
	pipelinesConf, err := serviceConf.Sub("pipelines")
	if err != nil {
		return err
	}

	templatesConf, err := conf.Sub("templates")
	if err != nil {
		return err
	}

	for templateInstanceID := range receiversMap {
		// Using a template requires the following format
		// receivers:
		//   template/template_type[/name]:
		//     parameter_one: value_one
		//     parameter_two: value_two
		templateInstanceIDParts := strings.SplitN(templateInstanceID, "/", 3)
		if templateInstanceIDParts[0] != "template" {
			continue
		}
		if len(templateInstanceIDParts) == 1 {
			return fmt.Errorf("'template/' must be followed by type")
		}
		templateType := templateInstanceIDParts[1]
		var templateInstanceName string
		if len(templateInstanceIDParts) == 3 {
			templateInstanceName = templateInstanceIDParts[2]
		}

		if !templatesConf.IsSet(templateType) {
			return fmt.Errorf("template type %q not found", templateType)
		}

		rawTemplate := templatesConf.Get(templateType).(string)
		var parsedTemplate *template.Template
		parsedTemplate, err = template.New(templateType).Parse(rawTemplate)
		if err != nil {
			return err
		}

		var parametersConf *confmap.Conf
		parametersConf, err = receiversConf.Sub(templateInstanceID)
		if err != nil {
			return fmt.Errorf("%q parameters: %w", templateInstanceID, err)
		}
		parameters := parametersConf.ToStringMap()

		var rendered bytes.Buffer
		if err = parsedTemplate.Execute(&rendered, parameters); err != nil {
			return fmt.Errorf("render template %q: %w", templateInstanceID, err)
		}

		var cfg TemplateConfig
		if err = yaml.Unmarshal(rendered.Bytes(), &cfg); err != nil {
			return fmt.Errorf("malformed template %q: %w", templateInstanceID, err)
		}

		if len(cfg.Receivers) == 0 {
			return fmt.Errorf("template %q: must have at least one receiver", templateInstanceID)
		}

		if len(cfg.Pipelines) == 0 {
			return fmt.Errorf("template %q: must have at least one pipeline", templateInstanceID)
		}

		// Apply a scope to all component IDs in the template.
		// At a minimum, the this appends the template type, but will
		// also apply the template instance name if possible.
		scopeMapKeys(cfg.Receivers, templateType, templateInstanceName)
		scopeMapKeys(cfg.Processors, templateType, templateInstanceName)
		for _, p := range cfg.Pipelines {
			scopeSliceValues(p.Receivers, templateType, templateInstanceName)
			scopeSliceValues(p.Processors, templateType, templateInstanceName)
		}

		// Update the receivers section by deleting the reference to the
		// template and replacing it with the rendered receiver(s).
		// Since this is a destructive change, we are updating a raw map
		// which we will then use when rebuilding the Conf.
		delete(receiversMap, templateInstanceID)
		for receiverID, receiverCfg := range cfg.Receivers {
			receiversMap[receiverID] = receiverCfg
		}

		// Update the processors section by adding any rendered processors.
		// Since this is an additive change, we are updating the original Conf.
		newProcessorsMap := make(map[string]any, len(cfg.Processors))
		for processorID, processorCfg := range cfg.Processors {
			newProcessorsMap[processorID] = processorCfg
		}
		if err = processorsConf.Merge(confmap.NewFromStringMap(newProcessorsMap)); err != nil {
			return err
		}

		// Add a dedicated forward connector for this template instance.
		// This will consume data from the partial pipelines and emit to
		// any top-level pipelines that used the template as a receiver.
		connectorID := "forward/" + templateType
		if templateInstanceName != "" {
			connectorID = connectorID + "/" + templateInstanceName
		}
		if err = connectorsConf.Merge(confmap.NewFromStringMap(
			map[string]any{connectorID: nil},
		)); err != nil {
			return err
		}

		// Crawl through existing "pipelines" and replace all references
		// to this instance of the template with the forward connector.
		// Since these are a 1:1 changes, we are updating the original Conf.
		//
		// Also take note of the pipeline data types in which the template is used.
		// We'll use these later to include only relevant partial pipelines.
		var usedTraces, usedMetrics, usedLogs bool
		pipelinesMap := pipelinesConf.ToStringMap()
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
			var pipelineConf *confmap.Conf
			pipelineConf, err = pipelinesConf.Sub(pipelineID)
			if err != nil {
				return err
			}
			pipelineMap := pipelineConf.ToStringMap()
			receiverIDs := pipelineMap["receivers"].([]any)
			for i, receiverID := range receiverIDs {
				if receiverID == templateInstanceID {
					receiverIDs[i] = connectorID
				}
			}
			if err = pipelinesConf.Merge(confmap.NewFromStringMap(
				map[string]any{
					pipelineID: map[string]any{
						"receivers": receiverIDs,
					},
				},
			)); err != nil {
				return err
			}
		}

		// For each partial pipeline, build a full pipeline by appending
		// the forward connector as the exporter.
		//
		// Only include the pipeline if the template was used as a receiver
		// in a pipeline of the same data type.
		newPipelines := make(map[string]any, len(cfg.Pipelines))
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
			if templateInstanceName != "" {
				scopedPipelineName += "/" + templateInstanceName
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
			if len(processors) > 0 {
				newPipeline["processors"] = processors
			}
			newPipelines[scopedPipelineName] = newPipeline
		}
		if len(newPipelines) == 0 {
			return fmt.Errorf("template %q: not used in supported pipeline type", templateInstanceID)
		}
		if err = pipelinesConf.Merge(confmap.NewFromStringMap(newPipelines)); err != nil {
			return err
		}
	}

	// At this point, all updates have been made. We need to rebuild the Conf
	// using the updated receivers map. All other changes were made to directly
	// to the original Conf so we can just copy those parts over.
	newConf := confmap.New()
	confMap := conf.ToStringMap()
	for k, v := range confMap {
		switch k {
		case "processors", "connectors":
			// Since these are optional sections, handle them separately below
		case "templates":
			// We don't want to include the templates section in the final Conf
		case "receivers":
			if err = newConf.Merge(confmap.NewFromStringMap(map[string]any{k: receiversMap})); err != nil {
				return err
			}
		case "service":
			if err = newConf.Merge(confmap.NewFromStringMap(map[string]any{k: v})); err != nil {
				return err
			}
			if err = newConf.Merge(confmap.NewFromStringMap(map[string]any{k: map[string]any{
				"pipelines": pipelinesConf.ToStringMap(),
			}})); err != nil {
				return err
			}
		default:
			if err = newConf.Merge(confmap.NewFromStringMap(map[string]any{k: v})); err != nil {
				return err
			}
		}
	}

	if err = newConf.Merge(confmap.NewFromStringMap(map[string]any{"connectors": connectorsConf.ToStringMap()})); err != nil {
		return err
	}
	if len(processorsConf.AllKeys()) > 0 {
		if err = newConf.Merge(confmap.NewFromStringMap(map[string]any{"processors": processorsConf.ToStringMap()})); err != nil {
			return err
		}
	}

	*conf = *newConf
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
