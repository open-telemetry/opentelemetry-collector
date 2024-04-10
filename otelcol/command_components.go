// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
)

type componentWithStability struct {
	Name      component.Type
	Stability map[string]string
}

type componentsOutput struct {
	BuildInfo  component.BuildInfo
	Receivers  []componentWithStability
	Processors []componentWithStability
	Exporters  []componentWithStability
	Connectors []componentWithStability
	Extensions []componentWithStability
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
					Name: con.Type(),
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
					Name: ext.Type(),
					Stability: map[string]string{
						"extension": ext.ExtensionStability().String(),
					},
				})
			}
			for _, prs := range sortFactoriesByType[processor.Factory](factories.Processors) {
				components.Processors = append(components.Processors, componentWithStability{
					Name: prs.Type(),
					Stability: map[string]string{
						"logs":    prs.LogsProcessorStability().String(),
						"metrics": prs.MetricsProcessorStability().String(),
						"traces":  prs.TracesProcessorStability().String(),
					},
				})
			}
			for _, rcv := range sortFactoriesByType[receiver.Factory](factories.Receivers) {
				components.Receivers = append(components.Receivers, componentWithStability{
					Name: rcv.Type(),
					Stability: map[string]string{
						"logs":    rcv.LogsReceiverStability().String(),
						"metrics": rcv.MetricsReceiverStability().String(),
						"traces":  rcv.TracesReceiverStability().String(),
					},
				})
			}
			for _, exp := range sortFactoriesByType[exporter.Factory](factories.Exporters) {
				components.Exporters = append(components.Exporters, componentWithStability{
					Name: exp.Type(),
					Stability: map[string]string{
						"logs":    exp.LogsExporterStability().String(),
						"metrics": exp.MetricsExporterStability().String(),
						"traces":  exp.TracesExporterStability().String(),
					},
				})
			}
			components.BuildInfo = set.BuildInfo
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
