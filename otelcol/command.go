// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"errors"
	"flag"

	"github.com/spf13/cobra"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/featuregates"
)

// NewCommand constructs a new cobra.Command using the given CollectorSettings.
// Any URIs specified in CollectorSettings.ConfigProviderSettings.ResolverSettings.URIs
// are considered defaults and will be overwritten by config flags passed as
// command-line arguments to the executable.
//
// Deprecated: [v0.103.0] use NewCommandMustSetProvider instead
func NewCommand(set CollectorSettings) *cobra.Command {
	return commandHelper(set, false)
}

// NewCommandMustSetProvider constructs a new cobra.Command using the given CollectorSettings.
// Any URIs specified in CollectorSettings.ConfigProviderSettings.ResolverSettings.URIs
// are considered defaults and will be overwritten by config flags passed as
// command-line arguments to the executable.
// At least one Provider must be supplied via CollectorSettings.ConfigProviderSettings.ResolverSettings.ProviderFactories.
func NewCommandMustSetProvider(set CollectorSettings) *cobra.Command {
	return commandHelper(set, true)
}

func commandHelper(set CollectorSettings, enforceProviders bool) *cobra.Command {
	flagSet := flags(featuregate.GlobalRegistry())
	rootCmd := &cobra.Command{
		Use:          set.BuildInfo.Command,
		Version:      set.BuildInfo.Version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := updateSettingsUsingFlags(&set, flagSet, enforceProviders)
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
	rootCmd.AddCommand(newValidateSubCommand(set, flagSet, enforceProviders))
	rootCmd.Flags().AddGoFlagSet(flagSet)
	return rootCmd
}

// Puts command line flags from flags into the CollectorSettings, to be used during config resolution.
func updateSettingsUsingFlags(set *CollectorSettings, flags *flag.FlagSet, enforceProviders bool) error {
	resolverSet := &set.ConfigProviderSettings.ResolverSettings
	configFlags := getConfigFlag(flags)

	if len(configFlags) > 0 {
		resolverSet.URIs = configFlags
	}
	if len(resolverSet.URIs) == 0 {
		return errors.New("at least one config flag must be provided")
	}

	if featuregates.UseUnifiedEnvVarExpansionRules.IsEnabled() && set.ConfigProviderSettings.ResolverSettings.DefaultScheme == "" {
		set.ConfigProviderSettings.ResolverSettings.DefaultScheme = "env"
	}

	// Provide a default set of providers and converters if none have been specified.
	// TODO: Remove this after CollectorSettings.ConfigProvider is removed and instead
	// do it in the builder.
	if len(resolverSet.ProviderFactories) == 0 && len(resolverSet.ConverterFactories) == 0 {
		if enforceProviders {
			return errors.New("at least one Provider must be supplied")
		}
		set.ConfigProviderSettings = newDefaultConfigProviderSettings(resolverSet.URIs)
	}
	return nil
}
