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
	"errors"
	"flag"

	"github.com/spf13/cobra"

	"go.opentelemetry.io/collector/featuregate"
)

// NewCommand constructs a new cobra.Command using the given CollectorSettings.
func NewCommand(set CollectorSettings) *cobra.Command {
	flagSet := flags(featuregate.GlobalRegistry())
	rootCmd := &cobra.Command{
		Use:          set.BuildInfo.Command,
		Version:      set.BuildInfo.Version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			col, err := newCollectorWithFlags(set, flagSet)
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

func newCollectorWithFlags(set CollectorSettings, flags *flag.FlagSet) (*Collector, error) {
	if set.ConfigProvider == nil {
		configFlags := getConfigFlag(flags)
		if len(configFlags) == 0 {
			return nil, errors.New("at least one config flag must be provided")
		}

		var err error
		set.ConfigProvider, err = NewConfigProvider(newDefaultConfigProviderSettings(configFlags))
		if err != nil {
			return nil, err
		}
	}
	return NewCollector(set)
}
