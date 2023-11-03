// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/component"
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
		RunE: func(cmd *cobra.Command, args []string) error {

			factories, err := set.Factories()
			if err != nil {
				return fmt.Errorf("failed to initialize factories: %w", err)
			}

			components := componentsOutput{}
			for con := range factories.Connectors {
				components.Connectors = append(components.Connectors, componentWithStability{
					Name: con,
					Stability: map[string]string{
						"logs-to-logs":    factories.Connectors[con].LogsToLogsStability().String(),
						"logs-to-metrics": factories.Connectors[con].LogsToMetricsStability().String(),
						"logs-to-traces":  factories.Connectors[con].LogsToTracesStability().String(),

						"metrics-to-logs":    factories.Connectors[con].MetricsToLogsStability().String(),
						"metrics-to-metrics": factories.Connectors[con].MetricsToMetricsStability().String(),
						"metrics-to-traces":  factories.Connectors[con].MetricsToTracesStability().String(),

						"traces-to-logs":    factories.Connectors[con].TracesToLogsStability().String(),
						"traces-to-metrics": factories.Connectors[con].TracesToMetricsStability().String(),
						"traces-to-traces":  factories.Connectors[con].TracesToTracesStability().String(),
					},
				})
			}
			for ext := range factories.Extensions {
				components.Extensions = append(components.Extensions, componentWithStability{
					Name: ext,
					Stability: map[string]string{
						"extension": factories.Extensions[ext].ExtensionStability().String(),
					},
				})
			}
			for prs := range factories.Processors {
				components.Processors = append(components.Processors, componentWithStability{
					Name: prs,
					Stability: map[string]string{
						"logs":    factories.Processors[prs].LogsProcessorStability().String(),
						"metrics": factories.Processors[prs].MetricsProcessorStability().String(),
						"traces":  factories.Processors[prs].TracesProcessorStability().String(),
					},
				})
			}
			for rcv := range factories.Receivers {
				components.Receivers = append(components.Receivers, componentWithStability{
					Name: rcv,
					Stability: map[string]string{
						"logs":    factories.Receivers[rcv].LogsReceiverStability().String(),
						"metrics": factories.Receivers[rcv].MetricsReceiverStability().String(),
						"traces":  factories.Receivers[rcv].TracesReceiverStability().String(),
					},
				})
			}
			for exp := range factories.Exporters {
				components.Exporters = append(components.Exporters, componentWithStability{
					Name: exp,
					Stability: map[string]string{
						"logs":    factories.Exporters[exp].LogsExporterStability().String(),
						"metrics": factories.Exporters[exp].MetricsExporterStability().String(),
						"traces":  factories.Exporters[exp].TracesExporterStability().String(),
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
