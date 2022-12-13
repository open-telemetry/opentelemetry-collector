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
	"flag"

	"github.com/spf13/cobra"
)

// newValidateSubCommand constructs a new validate sub command using the given CollectorSettings.
func newValidateSubCommand(set CollectorSettings, flagSet *flag.FlagSet) *cobra.Command {
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validates the config without running the collector",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

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
			return col.DryRun(cmd.Context())
		},
	}
	return validateCmd
}
