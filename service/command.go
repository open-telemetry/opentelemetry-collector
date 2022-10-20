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
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/config"

	"go.opentelemetry.io/collector/featuregate"
)

type componentsOutput struct {
	Version    string
	Receivers  []config.Type
	Processors []config.Type
	Exporters  []config.Type
	Extensions []config.Type
}

// newBuildCommand constructs a new cobra.Command sub command using the given CollectorSettings.
func newBuildSubCommand(set CollectorSettings) *cobra.Command {
	buildCmd := &cobra.Command{
		Use:   "build-info",
		Short: "Outputs available components in a given collector distribution",
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
			components.Version = set.BuildInfo.Version
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

// NewCommand constructs a new cobra.Command using the given CollectorSettings.
func NewCommand(set CollectorSettings) *cobra.Command {
	flagSet := flags()
	rootCmd := &cobra.Command{
		Use:          set.BuildInfo.Command,
		Version:      set.BuildInfo.Version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := featuregate.GetRegistry().Apply(gatesList); err != nil {
				return err
			}
			if set.ConfigProvider == nil {
				var err error

				configFlags := getConfigFlag(flagSet)
				if len(configFlags) == 0 {
					return errors.New("at least one config flag must be provided")
				}

				set.ConfigProvider, err = NewConfigProvider(newDefaultConfigProviderSettings(configFlags))
				if err != nil {
					return err
				}
			}
			col, err := New(set)
			if err != nil {
				return err
			}
			return col.Run(cmd.Context())
		},
	}
	rootCmd.AddCommand(newBuildSubCommand(set))
	rootCmd.Flags().AddGoFlagSet(flagSet)
	return rootCmd
}
