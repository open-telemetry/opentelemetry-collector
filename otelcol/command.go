// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
	rootCmd.AddCommand(newComponentsCommand(set))
	rootCmd.AddCommand(newValidateSubCommand(set, flagSet))
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
