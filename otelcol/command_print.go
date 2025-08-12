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
	featuregate.WithRegisterDescription("if set to true, turns on the raw mode of the print-config command"),
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

// printRawConfig prints the raw configuration before validation
func printRawConfig(cmd *cobra.Command, set CollectorSettings, flagSet *flag.FlagSet, outputFormat string) error {
	if !printCommandFeatureFlag.IsEnabled() {
		return errors.New("raw mode is currently experimental, use the otelcol.printInitialConfig feature gate to enable this mode")
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
}

// printRedactedConfig prints the validated configuration with sensitive data redacted
func printRedactedConfig(cmd *cobra.Command, set CollectorSettings, flagSet *flag.FlagSet, outputFormat string) error {
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

	// Convert config back to map for output - this will redact sensitive values
	confMap := confmap.New()
	if err := confMap.Marshal(cfg); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return printConfigData(confMap.ToStringMap(), outputFormat)
}

// printUnredactedConfig prints the validated configuration with all sensitive data visible
func printUnredactedConfig(cmd *cobra.Command, set CollectorSettings, flagSet *flag.FlagSet, outputFormat string) error {
	// TODO: Implement unredacted mode - for now, this behaves the same as redacted
	// This would require additional work to bypass configopaque masking
	fmt.Fprintf(os.Stderr, "Warning: unredacted mode is not yet implemented, showing redacted output\n")
	return printRedactedConfig(cmd, set, flagSet, outputFormat)
}

// newPrintConfigSubCommand constructs a new print-config command using the given CollectorSettings.
func newPrintConfigSubCommand(set CollectorSettings, flagSet *flag.FlagSet) *cobra.Command {
	var outputFormat string
	var mode string

	cmd := &cobra.Command{
		Use:   "print-config",
		Short: "Prints the Collector's configuration in the specified mode",
		Long: `Prints the Collector's configuration with different levels of processing:

- raw: Shows the resolved configuration before validation (may contain sensitive values)
- redacted: Shows the validated configuration with component defaults and sensitive data redacted (default, safe for sharing)
- unredacted: Shows the validated configuration with all sensitive data visible (most dangerous, not yet implemented)

The raw mode is currently experimental and requires the otelcol.printInitialConfig feature gate.`,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch strings.ToLower(mode) {
			case "raw", "initial":
				return printRawConfig(cmd, set, flagSet, outputFormat)
			case "redacted":
				return printRedactedConfig(cmd, set, flagSet, outputFormat)
			case "unredacted":
				return printUnredactedConfig(cmd, set, flagSet, outputFormat)
			default:
				return fmt.Errorf("invalid mode %q. Valid modes are: raw, redacted, unredacted", mode)
			}
		},
	}

	formatHelp := fmt.Sprintf("Output format. Available: yaml,json (defaults to yaml)")
	cmd.Flags().StringVar(&outputFormat, "format", "yaml", formatHelp)

	modeHelp := fmt.Sprintf("Configuration processing mode. Available: raw,redacted,unredacted (defaults to redacted)")
	cmd.Flags().StringVar(&mode, "mode", "redacted", modeHelp)

	cmd.Flags().AddGoFlagSet(flagSet)
	return cmd
}
