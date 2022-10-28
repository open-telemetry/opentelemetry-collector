package service

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

type componentsOutput struct {
	BuildInfo  component.BuildInfo
	Receivers  []config.Type
	Processors []config.Type
	Exporters  []config.Type
	Extensions []config.Type
}

// newBuildSubCommand constructs a new cobra.Command sub command using the given CollectorSettings.
func newBuildSubCommand(set CollectorSettings) *cobra.Command {
	buildCmd := &cobra.Command{
		Use:   "build-info",
		Short: "Outputs available components in this collector distribution",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			components := componentsOutput{}
			for ext := range set.Factories.Extensions {
				components.Extensions = append(components.Extensions, ext)
			}
			for prs := range set.Factories.Processors {
				components.Processors = append(components.Processors, prs)
			}
			for rcv := range set.Factories.Receivers {
				components.Receivers = append(components.Receivers, rcv)
			}
			for exp := range set.Factories.Exporters {
				components.Exporters = append(components.Exporters, exp)
			}
			components.BuildInfo = set.BuildInfo
			yamlData, err := yaml.Marshal(components)
			if err != nil {
				return err
			}
			fmt.Println(string(yamlData))
			return nil
		},
	}
	return buildCmd
}
