// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	yaml "go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
)

type componentWithStability struct {
	Name      component.Type
	Module    string
	Stability map[string]string
}

type componentWithoutStability struct {
	Scheme string `yaml:",omitempty"`
	Module string
}

type componentsOutput struct {
	BuildInfo  component.BuildInfo
	Receivers  []componentWithStability
	Processors []componentWithStability
	Exporters  []componentWithStability
	Connectors []componentWithStability
	Extensions []componentWithStability
	Providers  []componentWithoutStability
	Converters []componentWithoutStability `yaml:",omitempty"`
}

// newComponentsCommand constructs a new components command using the given CollectorSettings.
func newComponentsCommand(set CollectorSettings) *cobra.Command {
	return &cobra.Command{
		Use:   "components",
		Short: "Outputs available components in this collector distribution",
		Long:  "Outputs available components in this collector distribution including their stability levels. The output format is not stable and can change between releases.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			factories, err := set.Factories()
			if err != nil {
				return fmt.Errorf("failed to initialize factories: %w", err)
			}

			components := componentsOutput{}
			for _, con := range sortFactoriesByType[connector.Factory](factories.Connectors) {
				components.Connectors = append(components.Connectors, componentWithStability{
					Name:   con.Type(),
					Module: factories.ConnectorModules[con.Type()],
					Stability: map[string]string{
						"logs-to-logs":    con.LogsToLogsStability().String(),
						"logs-to-metrics": con.LogsToMetricsStability().String(),
						"logs-to-traces":  con.LogsToTracesStability().String(),

						"metrics-to-logs":    con.MetricsToLogsStability().String(),
						"metrics-to-metrics": con.MetricsToMetricsStability().String(),
						"metrics-to-traces":  con.MetricsToTracesStability().String(),

						"traces-to-logs":    con.TracesToLogsStability().String(),
						"traces-to-metrics": con.TracesToMetricsStability().String(),
						"traces-to-traces":  con.TracesToTracesStability().String(),
					},
				})
			}
			for _, ext := range sortFactoriesByType[extension.Factory](factories.Extensions) {
				components.Extensions = append(components.Extensions, componentWithStability{
					Name:   ext.Type(),
					Module: factories.ExtensionModules[ext.Type()],
					Stability: map[string]string{
						"extension": ext.Stability().String(),
					},
				})
			}
			for _, prs := range sortFactoriesByType[processor.Factory](factories.Processors) {
				components.Processors = append(components.Processors, componentWithStability{
					Name:   prs.Type(),
					Module: factories.ProcessorModules[prs.Type()],
					Stability: map[string]string{
						"logs":    prs.LogsStability().String(),
						"metrics": prs.MetricsStability().String(),
						"traces":  prs.TracesStability().String(),
					},
				})
			}
			for _, rcv := range sortFactoriesByType[receiver.Factory](factories.Receivers) {
				components.Receivers = append(components.Receivers, componentWithStability{
					Name:   rcv.Type(),
					Module: factories.ReceiverModules[rcv.Type()],
					Stability: map[string]string{
						"logs":    rcv.LogsStability().String(),
						"metrics": rcv.MetricsStability().String(),
						"traces":  rcv.TracesStability().String(),
					},
				})
			}
			for _, exp := range sortFactoriesByType[exporter.Factory](factories.Exporters) {
				components.Exporters = append(components.Exporters, componentWithStability{
					Name:   exp.Type(),
					Module: factories.ExporterModules[exp.Type()],
					Stability: map[string]string{
						"logs":    exp.LogsStability().String(),
						"metrics": exp.MetricsStability().String(),
						"traces":  exp.TracesStability().String(),
					},
				})
			}
			components.BuildInfo = set.BuildInfo

			for providerScheme, providerModuleModule := range set.ProviderModules {
				components.Providers = append(components.Providers, componentWithoutStability{
					Scheme: providerScheme,
					Module: providerModuleModule,
				})
			}

			for _, converterModule := range set.ConverterModules {
				components.Converters = append(components.Converters, componentWithoutStability{
					Module: converterModule,
				})
			}

			yamlData, err := yaml.Marshal(components)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), string(yamlData))
			return nil
		},
	}
}

func sortFactoriesByType[T component.Factory](factories map[component.Type]T) []T {
	// Gather component types (factories map keys)
	componentTypes := make([]component.Type, 0, len(factories))
	for componentType := range factories {
		componentTypes = append(componentTypes, componentType)
	}

	// Sort component types as strings
	sort.Slice(componentTypes, func(i, j int) bool {
		return componentTypes[i].String() < componentTypes[j].String()
	})

	// Build and return list of factories, sorted by component types
	sortedFactories := make([]T, 0, len(factories))
	for _, componentType := range componentTypes {
		sortedFactories = append(sortedFactories, factories[componentType])
	}

	return sortedFactories
}
