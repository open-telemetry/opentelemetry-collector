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

	"github.com/spf13/cobra"

	"go.opentelemetry.io/collector/featuregate"
)

// NewCommand constructs a new cobra.Command using the given CollectorSettings.
func NewCommand(set CollectorSettings) *cobra.Command {
	flagSet := flags()
	rootCmd := &cobra.Command{
		Use:          set.BuildInfo.Command,
		Version:      set.BuildInfo.Version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			set.DryRun = getDryRunFlag(flagSet)
			if err := featuregate.GetRegistry().Apply(getFeatureGatesFlag(flagSet)); err != nil {
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

	rootCmd.Flags().AddGoFlagSet(flagSet)
	return rootCmd
}
