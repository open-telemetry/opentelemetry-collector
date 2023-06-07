// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

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
			col, err := NewCollector(set)
			if err != nil {
				return err
			}
			return col.DryRun(cmd.Context())
		},
	}
	validateCmd.Flags().AddGoFlagSet(flagSet)
	return validateCmd
}
