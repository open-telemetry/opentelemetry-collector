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

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/component"
)

type componentsOutput struct {
	BuildInfo  component.BuildInfo
	Receivers  []component.Type
	Processors []component.Type
	Exporters  []component.Type
	Extensions []component.Type
}

// newBuildSubCommand constructs a new cobra.Command sub command using the given CollectorSettings.
func newBuildSubCommand(set CollectorSettings) *cobra.Command {
	buildCmd := &cobra.Command{
		Use:   "components",
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
			fmt.Fprint(cmd.OutOrStdout(), string(yamlData))
			return nil
		},
	}
	return buildCmd
}
