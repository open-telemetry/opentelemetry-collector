// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"errors"
	"flag"
	"fmt"

	"github.com/spf13/cobra"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/featuregate"
)

var printCommandFeatureFlag = featuregate.GlobalRegistry().MustRegister(
	"otelcol.printInitialConfig",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.120.0"),
	featuregate.WithRegisterDescription("if set to true, turns on the print-initial-config command"),
)

// newConfigPrintSubCommand constructs a new config print sub command using the given CollectorSettings.
func newConfigPrintSubCommand(set CollectorSettings, flagSet *flag.FlagSet) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "print-initial-config",
		Short: "Prints the Collector's configuration in YAML format after all config sources are resolved and merged",
		Long:  `Note: This command prints the final yaml configuration before it is unmarshaled into config structs, which may contain sensitive values.`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !printCommandFeatureFlag.IsEnabled() {
				return errors.New("print-initial-config is currently experimental, use the otelcol.printInitialConfig feature gate to enable this command")
			}
			err := updateSettingsUsingFlags(&set, flagSet)
			if err != nil {
				return err
			}
			resolver, err := confmap.NewResolver(set.ConfigProviderSettings.ResolverSettings)
			if err != nil {
				return fmt.Errorf("failed to create new resolver: %w", err)
			}
			conf, err := resolver.Resolve(cmd.Context())
			if err != nil {
				return fmt.Errorf("error while resolving config: %w", err)
			}
			b, err := yaml.Marshal(conf.ToStringMap())
			if err != nil {
				return fmt.Errorf("error while marshaling to YAML: %w", err)
			}
			fmt.Printf("%s\n", b)
			return nil
		},
	}
	cmd.Flags().AddGoFlagSet(flagSet)
	return cmd
}
