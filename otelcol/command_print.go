// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	yaml "go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/featuregate"
)

var printCommandFeatureFlag = featuregate.GlobalRegistry().MustRegister(
	"otelcol.printInitialConfig",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.120.0"),
	featuregate.WithRegisterDescription("if set to true, turns on the print-initial-config command"),
)

// printConfigData formats and prints configuration data using available configuration providers
func printConfigData(data map[string]any, format string) error {
	if format == "" {
		format = "yaml"
	}

	// Handle JSON output using standard library
	if strings.EqualFold(format, "json") {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	}

	if strings.EqualFold(format, "yaml") {
		b, err := yaml.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", b)
		return nil
	}
	return fmt.Errorf("Unrecognized print-format value: %s", format)
}

// newPrintInitialConfigSubCommand constructs a new config print sub command using the given CollectorSettings.
func newPrintInitialConfigSubCommand(set CollectorSettings, flagSet *flag.FlagSet) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "print-initial-config",
		Short: "Prints the Collector's configuration in YAML or JSON format after all config sources are resolved and merged",
		Long:  `Note: This command prints the final configuration before it is unmarshaled into config structs, which may contain sensitive values. Use --format to specify output format.`,
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
			return printConfigData(conf.ToStringMap(), outputFormat)
		},
	}

	formatHelp := fmt.Sprintf("Output format. Available: yaml,json (defaults to yaml)")
	cmd.Flags().StringVar(&outputFormat, "format", "yaml", formatHelp)
	cmd.Flags().AddGoFlagSet(flagSet)
	return cmd
}

// newPrintTypedConfigSubCommand constructs a new validated config print sub command using the given CollectorSettings.
func newPrintTypedConfigSubCommand(set CollectorSettings, flagSet *flag.FlagSet) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "print-config",
		Short: "Prints the Collector's validated configuration with defaults resolved",
		Long:  `This command validates the configuration and prints it with component defaults applied. This output is suitable for sharing as it masks sensitive values.`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := updateSettingsUsingFlags(&set, flagSet)
			if err != nil {
				return err
			}

			// Create config provider
			configProvider, err := NewConfigProvider(set.ConfigProviderSettings)
			if err != nil {
				return fmt.Errorf("failed to create config provider: %w", err)
			}

			// Get factories and validate
			factories, err := set.Factories()
			if err != nil {
				return fmt.Errorf("failed to get factories: %w", err)
			}

			cfg, err := configProvider.Get(cmd.Context(), factories)
			if err != nil {
				return fmt.Errorf("failed to get config: %w", err)
			}

			// Convert config back to map for output
			confMap := confmap.New()
			if err := confMap.Marshal(cfg); err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}

			return printConfigData(confMap.ToStringMap(), outputFormat)
		},
	}

	formatHelp := fmt.Sprintf("Output format. Available: yaml,json (defaults to yaml)")
	cmd.Flags().StringVar(&outputFormat, "format", "yaml", formatHelp)
	cmd.Flags().AddGoFlagSet(flagSet)
	return cmd
}
