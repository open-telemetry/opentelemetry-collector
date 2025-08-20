// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

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
	rootCmd.AddCommand(newFeatureGateCommand())
	rootCmd.AddCommand(newComponentsCommand(set))
	rootCmd.AddCommand(newValidateSubCommand(set, flagSet))
	rootCmd.AddCommand(newConfigPrintSubCommand(set, flagSet))
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

func newFeatureGateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "featuregate [feature-id]",
		Short: "Display feature gates information",
		Long:  "Display information about available feature gates and their status",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				found := false
				featuregate.GlobalRegistry().VisitAll(func(g *featuregate.Gate) {
					if g.ID() == args[0] {
						found = true
						fmt.Printf("Feature: %s\n", g.ID())
						fmt.Printf("Enabled: %v\n", g.IsEnabled())
						fmt.Printf("Stage: %s\n", g.Stage())
						fmt.Printf("Description: %s\n", g.Description())
						fmt.Printf("From Version: %s\n", g.FromVersion())
						if g.ToVersion() != "" {
							fmt.Printf("To Version: %s\n", g.ToVersion())
						}
					}
				})
				if !found {
					return fmt.Errorf("feature %q not found", args[0])
				}
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "ID\tEnabled\tStage\tDescription\n")
			featuregate.GlobalRegistry().VisitAll(func(g *featuregate.Gate) {
				fmt.Fprintf(w, "%s\t%v\t%s\t%s\n",
					g.ID(),
					g.IsEnabled(),
					g.Stage(),
					g.Description())
			})
			return w.Flush()
		},
	}
}
