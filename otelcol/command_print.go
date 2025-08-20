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
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/featuregate"
)

var printCommandFeatureFlag = featuregate.GlobalRegistry().MustRegister(
	"otelcol.printInitialConfig",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.120.0"),
	featuregate.WithRegisterDescription("if set to true, turns on the raw mode of the print-config command"),
)

// newConfigPrintSubCommand constructs a new print-config command using the given CollectorSettings.
func newConfigPrintSubCommand(set CollectorSettings, flagSet *flag.FlagSet) *cobra.Command {
	var outputFormat string
	var mode string

	cmd := &cobra.Command{
		Use:     "print-config",
		Aliases: []string{"print-initial-config"},
		Short:   "Prints the Collector's configuration in the specified mode",
		Long: `Prints the Collector's configuration with different levels of processing:

- redacted: Shows the resolved configuration with sensitive data redacted (default)
- validated: Shows the resolved and validated configuration (shows only valid configurations)
- unredacted: Shows the validated configuration with all sensitive data visible (shows complete data)

All modes are experimental requiring otelcol.printInitialConfig feature gate.`,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return configPrintSubCommand(cmd, set, flagSet, mode, outputFormat)
		},
	}

	formatHelp := fmt.Sprintf("Output format. Available: yaml,json (defaults to yaml)")
	cmd.Flags().StringVar(&outputFormat, "format", "yaml", formatHelp)

	modeHelp := fmt.Sprintf("Configuration processing mode. Available: initial,raw,redacted,unredacted (defaults to redacted)")
	cmd.Flags().StringVar(&mode, "mode", "redacted", modeHelp)

	cmd.Flags().AddGoFlagSet(flagSet)
	return cmd
}

func configPrintSubCommand(cmd *cobra.Command, set CollectorSettings, flagSet *flag.FlagSet, mode, outputFormat string) error {
	if !printCommandFeatureFlag.IsEnabled() {
		return errors.New("print-initial-config is currently experimental, use the otelcol.printInitialConfig feature gate to enable this command")
	}
	err := updateSettingsUsingFlags(&set, flagSet)
	if err != nil {
		return err
	}

	switch strings.ToLower(mode) {
	case "validated":
		return printRedactedConfig(cmd, set, true, outputFormat)
	case "redacted":
		fmt.Fprintf(os.Stderr, "Warning: redacted mode hides sensitive fields but does not validate. Use with caution.\n")
		return printRedactedConfig(cmd, set, false, outputFormat)
	case "unredacted":
		fmt.Fprintf(os.Stderr, "Warning: unredacted mode shows the complete configuration. Use with caution.\n")
		return printUnredactedConfig(cmd, set, outputFormat)
	default:
		return fmt.Errorf("invalid mode %q. Valid modes are: redacted, validated, unredacted", mode)
	}
}

// printConfigData formats and prints configuration data in yaml or json format.
func printConfigData(data map[string]any, format string) error {
	if format == "" {
		format = "yaml"
	}

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

func printConfiguration(cmd *cobra.Command, set CollectorSettings) (any, error) {
	factories, err := set.Factories()
	if err != nil {
		return nil, fmt.Errorf("failed to get factories: %w", err)
	}

	configProvider, err := NewConfigProvider(set.ConfigProviderSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create config provider: %w", err)
	}

	cfg, err := configProvider.Get(cmd.Context(), factories)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return cfg, nil
}

// printUnredactedConfig prints resolved configuration before interpreting
// with the intended types for each component, thus it shows the full
// configuration without considering configuopaque. Use with caution.
func printUnredactedConfig(cmd *cobra.Command, set CollectorSettings, outputFormat string) error {
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

// printRedactedConfig prints resolved configuration with its assigned
// types, but without validation. Notably, configopaque strings are printed
// as "[redacted]". This is the default.
func printRedactedConfig(cmd *cobra.Command, set CollectorSettings, validate bool, outputFormat string) error {
	cfg, err := printConfiguration(cmd, set)
	if err != nil {
		return err
	}

	if validate {
		if err = xconfmap.Validate(cfg); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
	}

	confMap := confmap.New()
	if err := confMap.Marshal(cfg); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return printConfigData(confMap.ToStringMap(), outputFormat)
}
