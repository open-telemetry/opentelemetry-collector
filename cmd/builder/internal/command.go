// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/builder/internal"

import (
	"fmt"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
	"go.opentelemetry.io/collector/cmd/builder/internal/config"
)

const (
	configFlag                 = "config"
	skipGenerateFlag           = "skip-generate"
	skipCompilationFlag        = "skip-compilation"
	skipGetModulesFlag         = "skip-get-modules"
	skipStrictVersioningFlag   = "skip-strict-versioning"
	ldflagsFlag                = "ldflags"
	gcflagsFlag                = "gcflags"
	distributionOutputPathFlag = "output-path"
	verboseFlag                = "verbose"
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
			cfg, err := initConfig(cmd.Flags())
			if err != nil {
				return err
			}
			if err = cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			if err = cfg.SetGoPath(); err != nil {
				return fmt.Errorf("go not found: %w", err)
			}

			if err = cfg.ParseModules(); err != nil {
				return fmt.Errorf("invalid module configuration: %w", err)
			}

			return builder.GenerateAndCompile(cfg)
		},
	}

	if err := initFlags(cmd.Flags()); err != nil {
		return nil, err
	}

	// version of this binary
	cmd.AddCommand(versionCommand())

	return cmd, nil
}

func initFlags(flags *flag.FlagSet) error {
	flags.String(configFlag, "", "build configuration file")
	// the distribution parameters, which we accept as CLI flags as well
	flags.Bool(skipGenerateFlag, false, "Whether builder should skip generating go code (default false)")
	flags.Bool(skipCompilationFlag, false, "Whether builder should only generate go code with no compile of the collector (default false)")
	flags.Bool(skipGetModulesFlag, false, "Whether builder should skip updating go.mod and retrieve Go module list (default false)")
	flags.Bool(skipStrictVersioningFlag, true, "Whether builder should skip strictly checking the calculated versions following dependency resolution")
	flags.Bool(verboseFlag, false, "Whether builder should print verbose output (default false)")
	flags.String(ldflagsFlag, "", `ldflags to include in the "go build" command`)
	flags.String(gcflagsFlag, "", `gcflags to include in the "go build" command`)
	flags.String(distributionOutputPathFlag, "", "Where to write the resulting files")
	return flags.MarkDeprecated(distributionOutputPathFlag, "use config distribution::output_path")
}

func initConfig(flags *flag.FlagSet) (*builder.Config, error) {
	cfg, err := builder.NewDefaultConfig()
	if err != nil {
		return nil, err
	}

	cfg.Logger.Info("OpenTelemetry Collector Builder", zap.String("version", version))

	var provider koanf.Provider
	cfgFile, _ := flags.GetString(configFlag)
	if cfgFile != "" {
		cfg.Logger.Info("Using config file", zap.String("path", cfgFile))
		// load the config file
		provider = file.Provider(cfgFile)
	} else {
		cfg.Logger.Info("Using default build configuration")
		// or the default if the config isn't provided
		provider = config.DefaultProvider()
	}

	k := koanf.New(".")
	if err = k.Load(provider, yaml.Parser()); err != nil {
		return nil, fmt.Errorf("failed to load configuration file: %w", err)
	}

	// handle env variables
	if err = k.Load(env.Provider(".", env.Opt{}), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	if err = k.UnmarshalWithConf("", cfg, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if err = applyFlags(flags, cfg); err != nil {
		return nil, fmt.Errorf("failed to apply flags configuration: %w", err)
	}

	return cfg, nil
}

func applyFlags(flags *flag.FlagSet, cfg *builder.Config) error {
	var errs, err error
	cfg.SkipGenerate, err = flags.GetBool(skipGenerateFlag)
	errs = multierr.Append(errs, err)
	cfg.SkipCompilation, err = flags.GetBool(skipCompilationFlag)
	errs = multierr.Append(errs, err)
	cfg.SkipGetModules, err = flags.GetBool(skipGetModulesFlag)
	errs = multierr.Append(errs, err)
	cfg.SkipStrictVersioning, err = flags.GetBool(skipStrictVersioningFlag)
	errs = multierr.Append(errs, err)

	if flags.Changed(ldflagsFlag) {
		cfg.LDSet = true
		cfg.LDFlags, err = flags.GetString(ldflagsFlag)
		errs = multierr.Append(errs, err)
	}
	if flags.Changed(gcflagsFlag) {
		cfg.GCSet = true
		cfg.GCFlags, err = flags.GetString(gcflagsFlag)
		errs = multierr.Append(errs, err)
	}

	cfg.Verbose, err = flags.GetBool(verboseFlag)
	errs = multierr.Append(errs, err)

	if flags.Changed(distributionOutputPathFlag) {
		cfg.Distribution.OutputPath, err = flags.GetString(distributionOutputPathFlag)
		errs = multierr.Append(errs, err)
	}

	return errs
}
