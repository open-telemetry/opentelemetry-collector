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

package otelcol // import "go.opentelemetry.io/collector/cmd/otelcol"

import (
	"github.com/spf13/cobra"

	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/featuregate"
)

// NewCobraCommand constructs a new cobra.Command using the given service.CollectorSettings.
func NewCobraCommand(set service.CollectorSettings) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          set.BuildInfo.Command,
		Version:      set.BuildInfo.Version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			featuregate.Apply(featuregate.GetFlags())
			if set.ConfigProvider == nil {
				set.ConfigProvider = service.MustNewDefaultConfigProvider(getConfigFlag(), getSetFlag())
			}
			col, err := service.New(set)
			if err != nil {
				return err
			}
			return col.Run(cmd.Context())
		},
	}

	rootCmd.Flags().AddGoFlagSet(flags())
	return rootCmd
}
