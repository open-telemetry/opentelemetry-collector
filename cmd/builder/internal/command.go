// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/builder/internal"

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
	"go.opentelemetry.io/collector/cmd/builder/internal/config"
)

const (
	skipGenerateFlag           = "skip-generate"
	skipCompilationFlag        = "skip-compilation"
	skipGetModulesFlag         = "skip-get-modules"
	skipStrictVersioningFlag   = "skip-strict-versioning"
	ldflagsFlag                = "ldflags"
	distributionOutputPathFlag = "output-path"
	verboseFlag                = "verbose"
)

var (
	cfgFile string
	cfg     = builder.NewDefaultConfig()
	k       = koanf.New(".")
)

// Command is the main entrypoint for this application
func Command() (*cobra.Command, error) {
	cmd := &cobra.Command{
		SilenceUsage:  true, // Don't print usage on Run error.
		SilenceErrors: true, // Don't print errors; main does it.
		Use:           "ocb",
		Long: fmt.Sprintf("OpenTelemetry Collector Builder (%s)", version) + `

ocb generates a custom OpenTelemetry Collector binary using the
build configuration given by the "--config" argument. If no build
configuration is provided, ocb will generate a default Collector.
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := initConfig(cmd.Flags()); err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			if err := cfg.SetGoPath(); err != nil {
				return fmt.Errorf("go not found: %w", err)
			}

			if err := cfg.ParseModules(); err != nil {
				return fmt.Errorf("invalid module configuration: %w", err)
			}

			return builder.GenerateAndCompile(cfg)
		},
	}

	cmd.Flags().StringVar(&cfgFile, "config", "", "build configuration file")

	// the distribution parameters, which we accept as CLI flags as well
	cmd.Flags().BoolVar(&cfg.SkipGenerate, skipGenerateFlag, false, "Whether builder should skip generating go code (default false)")
	cmd.Flags().BoolVar(&cfg.SkipCompilation, skipCompilationFlag, false, "Whether builder should only generate go code with no compile of the collector (default false)")
	cmd.Flags().BoolVar(&cfg.SkipGetModules, skipGetModulesFlag, false, "Whether builder should skip updating go.mod and retrieve Go module list (default false)")
	cmd.Flags().BoolVar(&cfg.SkipStrictVersioning, skipStrictVersioningFlag, true, "Whether builder should skip strictly checking the calculated versions following dependency resolution")
	cmd.Flags().BoolVar(&cfg.Verbose, verboseFlag, false, "Whether builder should print verbose output (default false)")
	cmd.Flags().StringVar(&cfg.LDFlags, ldflagsFlag, "", `ldflags to include in the "go build" command`)
	cmd.Flags().StringVar(&cfg.Distribution.OutputPath, distributionOutputPathFlag, cfg.Distribution.OutputPath, "Where to write the resulting files")
	if err := cmd.Flags().MarkDeprecated(distributionOutputPathFlag, "use config distribution::output_path"); err != nil {
		return nil, err
	}
	// version of this binary
	cmd.AddCommand(versionCommand())

	return cmd, nil
}

func initConfig(flags *flag.FlagSet) error {
	cfg.Logger.Info("OpenTelemetry Collector Builder",
		zap.String("version", version))

	var provider koanf.Provider

	if cfgFile != "" {
		// load the config file
		provider = file.Provider(cfgFile)
	} else {
		// or the default if the config isn't provided
		provider = config.DefaultProvider()
		cfg.Logger.Info("Using default build configuration")
	}

	if err := k.Load(provider, yaml.Parser()); err != nil {
		return fmt.Errorf("failed to load configuration file: %w", err)
	}

	// handle env variables
	if err := k.Load(env.Provider("", ".", func(s string) string {
		// Only values from the `dist.` group can be set,
		// and the subfields in `dist.` contain `_` in their names.
		// All other fields are arrays and the koanf env provider doesn't provide a straightforward way to set arrays.
		return strings.Replace(strings.ToLower(s), "dist_", "dist.", 1)
	}), nil); err != nil {
		return fmt.Errorf("failed to load environment variables: %w", err)
	}

	cfgFromFile := builder.Config{}
	if err := k.UnmarshalWithConf("", &cfgFromFile, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	applyCfgFromFile(flags, cfgFromFile)

	if cfgFile != "" {
		cfg.Logger.Info("Using config file", zap.String("path", cfgFile))
	}

	return nil
}

func applyCfgFromFile(flags *flag.FlagSet, cfgFromFile builder.Config) {
	cfg.Exporters = cfgFromFile.Exporters
	cfg.Extensions = cfgFromFile.Extensions
	cfg.Receivers = cfgFromFile.Receivers
	cfg.Processors = cfgFromFile.Processors
	cfg.Connectors = cfgFromFile.Connectors
	cfg.Providers = cfgFromFile.Providers
	cfg.Replaces = cfgFromFile.Replaces
	cfg.Excludes = cfgFromFile.Excludes

	cfg.ConfResolver.DefaultURIScheme = cfgFromFile.ConfResolver.DefaultURIScheme

	if !flags.Changed(distributionOutputPathFlag) && cfgFromFile.Distribution.OutputPath != "" {
		cfg.Distribution.OutputPath = cfgFromFile.Distribution.OutputPath
	}

	cfg.Distribution.Name = cfgFromFile.Distribution.Name
	cfg.Distribution.Description = cfgFromFile.Distribution.Description
	cfg.Distribution.Version = cfgFromFile.Distribution.Version
	cfg.Distribution.OtelColVersion = cfgFromFile.Distribution.OtelColVersion
	cfg.Distribution.Go = cfgFromFile.Distribution.Go
	cfg.Distribution.Module = cfgFromFile.Distribution.Module
	cfg.Distribution.BuildTags = cfgFromFile.Distribution.BuildTags
	cfg.Distribution.DebugCompilation = cfgFromFile.Distribution.DebugCompilation
}
