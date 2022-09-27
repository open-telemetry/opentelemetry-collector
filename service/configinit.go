// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service // import "go.opentelemetry.io/collector/service"

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/service/internal/components"
)

const (
	sampleConfigHeader = `Opentelemetry Collector sample configuration. Double check the pipeline configuration
by ensuring the desired components belong to the respective pipeline type(ex metrics, logs, traces).`
)

type configCommand struct {
	exiter    func(args ...error) error
	stdio     terminal.Stdio
	available *componentsAvailable
}

type componentsAvailable struct {
	exporters  []string
	processors []string
	receivers  []string
	extensions []string
	pipelines  []string
}

type optionsSelected struct {
	Exporters  []string
	Processors []string
	Receivers  []string
	Extensions []string
	Pipelines  []string

	ConfigFileName string
	DryRun         bool
}

var defaultConfigFileName = "otelcol.config.yaml"

var cfgCmd *configCommand
var log *zap.Logger
var opts = new(optionsSelected)

func init() {
	var err error
	log, err = zap.NewDevelopment(zap.AddStacktrace(zap.DPanicLevel))
	if err != nil {
		panic(fmt.Sprintf("failed to obtain a logger instance: %v", err))
	}
}

// NewConfigurationInitCommand creates a new command for creating sample configurations
func NewConfigurationInitCommand(set CollectorSettings) *cobra.Command {
	cfgCmd = &configCommand{
		exiter: func(args ...error) error {
			log.Error(fmt.Sprintf("command failed %v", args))
			return nil
		},
		stdio:     terminal.Stdio{In: os.Stdin, Out: os.Stdout, Err: os.Stderr},
		available: getAvailableComponentsForPrompts(set.Factories),
	}
	opts = &optionsSelected{}
	cmd := &cobra.Command{
		Use:          "config",
		Short:        "Generate a sample configuration file.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// no flags were passed(except dry run), show prompt
			if (len(os.Args) == 3 && cmd.Flags().Lookup("dry-run").Changed) || len(os.Args) <= 2 {
				err := showPrompt(cfgCmd.available, opts, cfgCmd.stdio)
				// err could be interrupt or error occurred due to failed write to the answer object,
				// in either case, we can't proceed further
				if err != nil {
					return cfgCmd.exiter(err)
				}
			}

			exportersNode, err := getExportersNode(set.Factories, opts.Exporters)
			if err != nil {
				return cfgCmd.exiter(err)
			}
			processorsNode, err := getProcessorsNode(set.Factories, opts.Processors)
			if err != nil {
				return cfgCmd.exiter(err)
			}
			receiversNode, err := getReceiversNode(set.Factories, opts.Receivers)
			if err != nil {
				return cfgCmd.exiter(err)
			}
			extensionsNode, err := getExtensionsNode(set.Factories, opts.Extensions)
			if err != nil {
				return cfgCmd.exiter(err)
			}

			serviceNode, err := getServiceNode(set.Factories, opts.Pipelines)
			if err != nil {
				return cfgCmd.exiter(err)
			}

			configYaml := getSampleConfig(exportersNode, processorsNode, receiversNode, extensionsNode, serviceNode)
			configBuf, err := getConfigurationBuffer(set.Factories, configYaml)
			if err != nil {
				return cfgCmd.exiter(err)
			}
			err = outputConfiguration(configBuf)
			if err != nil {
				return cfgCmd.exiter(err)
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringArrayVarP(&opts.Exporters, "exporter", "e", []string{}, "Exporters to be configured.")
	cmd.PersistentFlags().StringArrayVarP(&opts.Processors, "processor", "p", []string{}, "Processors to be configured.")
	cmd.PersistentFlags().StringArrayVarP(&opts.Receivers, "receiver", "r", []string{}, "Receivers to be configured.")
	cmd.PersistentFlags().StringArrayVarP(&opts.Extensions, "extension", "x", []string{}, "Extensions to be configured.")
	cmd.PersistentFlags().StringArrayVarP(&opts.Pipelines, "pipeline", "l", []string{}, "Specify the pipelines to be configured.")
	cmd.PersistentFlags().BoolVarP(&opts.DryRun, "dry-run", "d", false, "Dry run.")
	cmd.PersistentFlags().StringVarP(&opts.ConfigFileName, "config-file", "f", defaultConfigFileName, "Name of the config file.")
	return cmd

}

func getAvailableComponentsForPrompts(factories component.Factories) *componentsAvailable {
	c := &componentsAvailable{}

	for _, exp := range factories.Exporters {
		c.exporters = append(c.exporters, string(exp.Type()))
	}
	for _, proc := range factories.Processors {
		c.processors = append(c.processors, string(proc.Type()))
	}
	for _, rec := range factories.Receivers {
		c.receivers = append(c.receivers, string(rec.Type()))
	}
	for _, ext := range factories.Extensions {
		c.extensions = append(c.extensions, string(ext.Type()))
	}
	c.pipelines = []string{string(config.MetricsDataType), string(config.TracesDataType), string(config.LogsDataType)}
	return c
}

// showPrompt displays a prompt to the user for configuration options using stdio terminal source
// and answers as output sink.
func showPrompt(c *componentsAvailable, answers interface{}, stdio terminal.Stdio) error {
	promptExp := &survey.MultiSelect{
		Message: "Select exporters",
		Options: c.exporters,
		Filter:  filterFunc,
	}
	promptProc := &survey.MultiSelect{
		Message: "Select processors",
		Options: c.processors,
		Filter:  filterFunc,
	}
	promptRec := &survey.MultiSelect{
		Message: "Select receivers",
		Options: c.receivers,
		Filter:  filterFunc,
	}
	promptExt := &survey.MultiSelect{
		Message: "Select extensions",
		Options: c.extensions,
		Filter:  filterFunc,
	}

	promptPipelines := &survey.MultiSelect{
		Message: "Select pipelines to enable",
		Options: []string{"metrics", "traces", "logs"},
	}

	promptFile := &survey.Input{
		Message: "Name of the config file",
		Default: defaultConfigFileName,
	}

	questions := []*survey.Question{
		{
			Name:     "Exporters",
			Prompt:   promptExp,
			Validate: survey.Required, // at least one exporter is needed
		},
		{
			Name:   "Processors",
			Prompt: promptProc,
		},
		{
			Name:     "Receivers",
			Prompt:   promptRec,
			Validate: survey.Required, // at least one receiver is required
		},
		{
			Name:   "Extensions",
			Prompt: promptExt,
		},
		{
			Name:   "Pipelines",
			Prompt: promptPipelines,
		},
	}
	if !opts.DryRun {
		questions = append(questions, &survey.Question{
			Name:   "ConfigFileName",
			Prompt: promptFile,
		})
	}
	return survey.Ask(questions, answers, survey.WithStdio(stdio.In, stdio.Out, stdio.Err))
}

// filterFunc filters out the options that containing the given prefix with len at least 3
func filterFunc(filterValue string, optValue string, optIndex int) bool {
	return strings.Contains(optValue, filterValue) && len(optValue) >= 3
}

func getSampleConfig(exportersNode, processorsNode, receiversNode, extensionsNode, serviceNode *yaml.Node) *yaml.Node {

	content := make([]*yaml.Node, 0)

	content = append(content, exportersNode.Content...)
	content = append(content, processorsNode.Content...)
	content = append(content, receiversNode.Content...)
	content = append(content, extensionsNode.Content...)
	content = append(content, serviceNode.Content...)

	configYaml := newDocumentNode(
		newMappingNode(content...),
	)

	return configYaml
}

func getSampleConfigForComponent(factories component.Factories, compType string, compName string) (string, error) {
	t := config.Type(compName)
	var f component.Factory
	switch compType {
	case components.ZapKindReceiver:
		f = factories.Receivers[t]
	case components.ZapKindProcessor:
		f = factories.Processors[t]
	case components.ZapKindExporter:
		f = factories.Exporters[t]
	case components.ZapKindExtension:
		f = factories.Extensions[t]
	}
	if f == nil {
		return "", fmt.Errorf("unknown %s name %q", compType, compName)
	}
	return f.SampleConfig(), nil
}

// getConfigurationBuffer outputs the configuration to a buffer
func getConfigurationBuffer(factories component.Factories, configYaml *yaml.Node) (*bytes.Buffer, error) {
	var b bytes.Buffer

	ymlEncoder := yaml.NewEncoder(&b)
	defer ymlEncoder.Close()

	ymlEncoder.SetIndent(2)
	err := ymlEncoder.Encode(configYaml)
	if err != nil {
		return nil, err
	}

	// check if the configuration is valid using yaml provider
	// err can be ignored
	cfgSet, _ := NewConfigProvider(ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:      []string{fmt.Sprintf("yaml:%s", b.String())},
			Providers: makeMapProvidersMap(yamlprovider.New()),
		},
	})

	// only interested in err
	_, err = cfgSet.Get(context.Background(), factories)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// outputConfiguration outputs the configuration to either the console(for dry-run) or to a file.
func outputConfiguration(b *bytes.Buffer) error {
	var writer = os.Stdout
	if !opts.DryRun {
		// create a new yaml file
		f, err := os.Create(filepath.Clean(opts.ConfigFileName))
		if err != nil {
			log.Fatal("error", zap.Error(err))
		}
		defer func() {
			f.Close()
			log.Info(fmt.Sprintf("Sample configuration created in %s", opts.ConfigFileName))
		}()
		writer = f
	}
	_, err := writer.Write(b.Bytes())
	return err
}

// getComponentNode create a yaml.Node from the given set of factories and the corresponding component type.
// For example, given the exporter factories and type exporter with [otlp, logging] as components, it will
// return an Node reprensentation equivalent of-
// exporters.otlp = {}
// exporters.logging = {}
func getComponentNode(factories component.Factories, components []string, componentType string) (*yaml.Node, error) {
	content := newMappingNode()

	for index := range components {
		yamlConfig, err := getSampleConfigForComponent(factories, componentType, components[index])
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(yamlConfig) == "" {
			keyVal := newSimpleScalarNode(components[index])
			keyVal.HeadComment = fmt.Sprintf("Sample config for %v %v is not implemented.", components[index], componentType)
			content.Content = append(content.Content, keyVal, newMappingNode())
			continue
		}
		var mapNode yaml.Node
		err = yaml.Unmarshal([]byte(yamlConfig), &mapNode)
		if err != nil {
			return nil, err
		}

		content.Content = append(content.Content, mapNode.Content[0].Content...)
	}
	return newMappingNode(
		// plural form of the component type
		newSimpleScalarNode(fmt.Sprintf("%ss", componentType)),
		content,
	), nil
}

func getExportersNode(factories component.Factories, exporters []string) (*yaml.Node, error) {
	return getComponentNode(factories, exporters, "exporter")
}

func getProcessorsNode(factories component.Factories, processors []string) (*yaml.Node, error) {
	return getComponentNode(factories, processors, "processor")
}

func getReceiversNode(factories component.Factories, receivers []string) (*yaml.Node, error) {
	return getComponentNode(factories, receivers, "receiver")
}

func getExtensionsNode(factories component.Factories, extensions []string) (*yaml.Node, error) {
	return getComponentNode(factories, extensions, "extension")
}

func getServiceNode(factories component.Factories, pipelines []string) (*yaml.Node, error) {

	acceptedPipelines := []config.DataType{config.MetricsDataType, config.TracesDataType, config.LogsDataType}
	pipelineSet := make(map[string]bool)
	pipeline := newMappingNode()
	for index := range pipelines {
		if !contains(acceptedPipelines, pipelines[index]) {
			return nil, fmt.Errorf("%s is not a valid pipeline", pipelines[index])
		}
		isPresent := pipelineSet[pipelines[index]]
		if isPresent {
			log.Warn(fmt.Sprintf("Pipeline %s is repeated. Skipping", pipelines[index]))
			continue
		}
		pipelineSet[pipelines[index]] = true
		pipelineType := newSimpleScalarNode(pipelines[index])

		// create exporters, processors, receivers and logs nodes
		pipelineComponents := newMappingNode(
			newScalarNodeWithComment("exporters", fmt.Sprintf("Specify exporters for the %s pipeline", pipelines[index])),
			getServiceExporterNode(factories, pipelines[index]),
			newScalarNodeWithComment("processors", fmt.Sprintf("Specify processors for the %s pipeline", pipelines[index])),
			getServiceProcessorNode(factories, pipelines[index]),
			newScalarNodeWithComment("receivers", fmt.Sprintf("Specify receivers for the %s pipeline", pipelines[index])),
			getServiceReceiverNode(factories, pipelines[index]),
		)
		pipeline.Content = append(pipeline.Content, pipelineType, pipelineComponents)
	}

	serviceNode := newMappingNode(
		newSimpleScalarNode("service"),
		newMappingNode(
			newScalarNodeWithComment("extensions", "Specify extensions for the collector"),
			getServiceExtensionNode(factories),
			newSimpleScalarNode("pipelines"),
			pipeline,
		),
	)
	return serviceNode, nil
}

func contains(acceptedPipelines []config.Type, s string) bool {
	for _, acceptedPipeline := range acceptedPipelines {
		if string(acceptedPipeline) == s {
			return true
		}
	}
	return false
}

func getServiceExporterNode(factories component.Factories, pipeline string) *yaml.Node {
	return getPipelineComponentsNode(factories, components.ZapKindExporter, opts.Exporters, pipeline)
}

func getServiceProcessorNode(factories component.Factories, pipeline string) *yaml.Node {
	return getPipelineComponentsNode(factories, components.ZapKindProcessor, opts.Processors, pipeline)
}

func getServiceReceiverNode(factories component.Factories, pipeline string) *yaml.Node {
	return getPipelineComponentsNode(factories, components.ZapKindReceiver, opts.Receivers, pipeline)
}

func getServiceExtensionNode(factories component.Factories) *yaml.Node {
	// extension does not belong to a pipeline
	return getPipelineComponentsNode(factories, components.ZapKindExtension, opts.Extensions, "")
}

func getPipelineComponentsNode(factories component.Factories, componentKind string, componentsSelected []string, pipeline string) *yaml.Node {
	var children []*yaml.Node
	for _, exp := range componentsSelected {
		sl := getStabilityLevel(factories, componentKind, exp, pipeline)
		// if the stability level is undefined, we assume it is not supporting the given pipeline
		if sl != component.StabilityLevelUndefined {
			children = append(children, newSimpleScalarNode(exp))
		}
	}
	return newSequenceNode(children...)
}

// getStabilityLevel returns the stability level for a component for a pipeline type(metrics, logs, traces)
func getStabilityLevel(factories component.Factories, compType string, compName string, pipeline string) component.StabilityLevel {
	switch compType {
	case components.ZapKindExporter:
		ef := factories.Exporters[config.Type(compName)]
		if pipeline == string(config.MetricsDataType) {
			return ef.MetricsExporterStability()
		}
		if pipeline == string(config.LogsDataType) {
			return ef.LogsExporterStability()
		}
		if pipeline == string(config.TracesDataType) {
			return ef.TracesExporterStability()
		}
	case components.ZapKindProcessor:
		pf := factories.Processors[config.Type(compName)]
		if pipeline == string(config.MetricsDataType) {
			return pf.MetricsProcessorStability()
		}
		if pipeline == string(config.LogsDataType) {
			return pf.LogsProcessorStability()
		}
		if pipeline == string(config.TracesDataType) {
			return pf.TracesProcessorStability()
		}
	case components.ZapKindReceiver:
		rf := factories.Receivers[config.Type(compName)]
		if pipeline == string(config.MetricsDataType) {
			return rf.MetricsReceiverStability()
		}
		if pipeline == string(config.LogsDataType) {
			return rf.LogsReceiverStability()
		}
		if pipeline == string(config.TracesDataType) {
			return rf.TracesReceiverStability()
		}
	case components.ZapKindExtension:
		ex := factories.Extensions[config.Type(compName)]
		return ex.ExtensionStability()
	}
	return component.StabilityLevelUndefined
}

// newDocumentNode creates a yaml node with kind yaml.DocumentNode
func newDocumentNode(children ...*yaml.Node) *yaml.Node {
	return &yaml.Node{
		Kind:        yaml.DocumentNode,
		Content:     children,
		HeadComment: sampleConfigHeader,
	}
}

// newSequenceNode creates a yaml node with kind yaml.SequenceNode
func newSequenceNode(content ...*yaml.Node) *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: content,
		Style:   yaml.FlowStyle,
	}
}

// newMappingNode creates a new mapping node with the given children.
func newMappingNode(children ...*yaml.Node) *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: children,
	}
}

// newScalarNode creates a yml node holding a scalar value
func newScalarNode(value string, doc string, style yaml.Style) *yaml.Node {
	return &yaml.Node{
		Kind:        yaml.ScalarNode,
		Value:       value,
		HeadComment: doc,
		Style:       style,
	}
}

// newSimpleScalarNode creates a scalar node with the given value
func newSimpleScalarNode(value string) *yaml.Node {
	return newScalarNode(value, "", yaml.FlowStyle)
}

// newScalarNodeWithComment creates a new scalar node with the given value and HeadComment
func newScalarNodeWithComment(value string, doc string) *yaml.Node {
	return newScalarNode(value, doc, yaml.FlowStyle)
}
