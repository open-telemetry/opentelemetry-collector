// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"flag"
	"log"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/confmap"
)

// newExamineSubCommand constructs a new examine sub command using the given CollectorSettings.
func newExamineSubCommand(set CollectorSettings, flagSet *flag.FlagSet) *cobra.Command {
	examineCmd := &cobra.Command{
		Use:   "examine",
		Short: "Logs the final configuration after all --config sources are resolved and merged",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := updateSettingsUsingFlags(&set, flagSet)
			if err != nil {
				return err
			}
			resolver, err := confmap.NewResolver(set.ConfigProviderSettings.ResolverSettings)
			if err != nil {
				return err
			}
			conf, err := resolver.Resolve(cmd.Context())
			if err != nil {
				return err
			}
			b, err := yaml.Marshal(conf.ToStringMap())
			if err != nil {
				return err
			}
			log.Printf("\n%s", b)
			return nil
		},
	}
	examineCmd.Flags().AddGoFlagSet(flagSet)
	return examineCmd
}
