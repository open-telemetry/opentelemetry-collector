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
// Any URIs specified in CollectorSettings.ConfigProviderSettings.ResolverSettings.URIs
// are considered defaults and will be overwritten by config flags passed as
// command-line arguments to the executable.
// At least one Provider must be set.
func NewCommand(set CollectorSettings) *cobra.Command {
	flagSet := flags(featuregate.GlobalRegistry())
	rootCmd := &cobra.Command{
		Use:          set.BuildInfo.Command,
		Version:      set.BuildInfo.Version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := updateSettingsUsingFlags(&set, flagSet)
			if err != nil {
				return err
			}

			col, err := NewCollector(set)
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

// Puts command line flags from flags into the CollectorSettings, to be used during config resolution.
func updateSettingsUsingFlags(set *CollectorSettings, flags *flag.FlagSet) error {
	resolverSet := &set.ConfigProviderSettings.ResolverSettings
	configFlags := getConfigFlag(flags)

	if len(configFlags) > 0 {
		resolverSet.URIs = configFlags
	}
	if len(resolverSet.URIs) == 0 {
		return errors.New("at least one config flag must be provided")
	}

	if set.ConfigProviderSettings.ResolverSettings.DefaultScheme == "" {
		set.ConfigProviderSettings.ResolverSettings.DefaultScheme = "env"
	}

	if len(resolverSet.ProviderFactories) == 0 {
		return errors.New("at least one Provider must be supplied")
	}
	return nil
}
