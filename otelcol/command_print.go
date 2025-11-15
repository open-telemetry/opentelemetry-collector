// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	yaml "go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/featuregate"
)

const featureGateName = "otelcol.printInitialConfig"

var printCommandFeatureFlag = featuregate.GlobalRegistry().MustRegister(
	featureGateName,
	featuregate.StageBeta,
	featuregate.WithRegisterFromVersion("v0.120.0"),
	featuregate.WithRegisterDescription("if set to true, enable the print-config command"),
)

// newConfigPrintSubCommand constructs a new print-config command using the given CollectorSettings.
func newConfigPrintSubCommand(set CollectorSettings, flagSet *flag.FlagSet) *cobra.Command {
	var outputFormat string
	var mode string
	var validate bool

	cmd := &cobra.Command{
		Use:     "print-config",
		Aliases: []string{"print-initial-config"},
		Short:   "Prints the Collector's configuration in the specified mode",
		Long: `Prints the Collector's configuration with different levels of processing:

- redacted: Shows the resolved configuration with sensitive data redacted (default)
- unredacted: Shows the resolved configuration with all sensitive data visible

The output prints in YAML by default. To print JSON use --format=json,
however this is considered unstable.

Validation is enabled by default, as a safety measure.

All modes are experimental, requiring the otelcol.printInitialConfig feature gate.`,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			pc := printContext{
				cmd:          cmd,
				stdout:       cmd.OutOrStdout(),
				set:          set,
				outputFormat: outputFormat,
				validate:     validate,
			}
			return pc.configPrintSubCommand(flagSet, mode)
		},
	}

	formatHelp := "Output format: yaml (default), json (unstable))"
	cmd.Flags().StringVar(&outputFormat, "format", "yaml", formatHelp)

	modeHelp := "Operating mode: redacted (default), unredacted"
	cmd.Flags().StringVar(&mode, "mode", "redacted", modeHelp)

	validateHelp := "Validation mode: true (default), false"
	cmd.Flags().BoolVar(&validate, "validate", true, validateHelp)

	cmd.Flags().AddGoFlagSet(flagSet)
	return cmd
}

type printContext struct {
	cmd          *cobra.Command
	stdout       io.Writer
	set          CollectorSettings
	outputFormat string
	validate     bool
}

func (pctx *printContext) configPrintSubCommand(flagSet *flag.FlagSet, mode string) error {
	if !printCommandFeatureFlag.IsEnabled() {
		return errors.New("print-config is currently experimental, use the otelcol.printInitialConfig feature gate to enable this command")
	}
	err := updateSettingsUsingFlags(&pctx.set, flagSet)
	if err != nil {
		return err
	}

	switch strings.ToLower(mode) {
	case "redacted":
		return pctx.printRedactedConfig()
	case "unredacted":
		return pctx.printUnredactedConfig()
	default:
		return fmt.Errorf("invalid mode %q: modes are: redacted, unredacted", mode)
	}
}

// printConfigData formats and prints configuration data in yaml or json format.
func (pctx *printContext) printConfigData(data map[string]any) error {
	format := pctx.outputFormat
	if format == "" {
		format = "yaml"
	}

	switch {
	case strings.EqualFold(format, "yaml"):
		b, err := yaml.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Fprintf(pctx.stdout, "%s\n", b)
		return nil

	case strings.EqualFold(format, "json"):
		encoder := json.NewEncoder(pctx.stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	}

	return fmt.Errorf("unrecognized print format: %s", format)
}

func (pctx *printContext) getPrintableConfig() (any, error) {
	var factories Factories
	if pctx.set.Factories != nil {
		var err error
		factories, err = pctx.set.Factories()
		if err != nil {
			return nil, fmt.Errorf("failed to get factories: %w", err)
		}
	}

	configProvider, err := NewConfigProvider(pctx.set.ConfigProviderSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create config provider: %w", err)
	}

	cfg, err := configProvider.Get(pctx.cmd.Context(), factories)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return cfg, nil
}

// printUnredactedConfig prints resolved configuration before interpreting
// with the intended types for each component, thus it shows the full
// configuration without considering configuopaque. Use with caution.
func (pctx *printContext) printUnredactedConfig() error {
	if pctx.validate {
		// Validation serves prevent revealing invalid data.
		cfg, err := pctx.getPrintableConfig()
		if err != nil {
			return err
		}
		if err = xconfmap.Validate(cfg); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}

		// Note: we discard the validated configuration.
	}
	resolver, err := confmap.NewResolver(pctx.set.ConfigProviderSettings.ResolverSettings)
	if err != nil {
		return fmt.Errorf("failed to create new resolver: %w", err)
	}
	conf, err := resolver.Resolve(pctx.cmd.Context())
	if err != nil {
		return fmt.Errorf("error while resolving config: %w", err)
	}
	return pctx.printConfigData(conf.ToStringMap())
}

// printRedactedConfig prints resolved configuration with its assigned
// types, but without validation. Notably, configopaque strings are printed
// as "[redacted]". This is the default.
func (pctx *printContext) printRedactedConfig() error {
	cfg, err := pctx.getPrintableConfig()
	if err != nil {
		return err
	}

	if pctx.validate {
		if err = xconfmap.Validate(cfg); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
	}

	confMap := confmap.New()
	if err := confMap.Marshal(cfg); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return pctx.printConfigData(confMap.ToStringMap())
}
