// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "go.opentelemetry.io/collector/cmd/builder/internal"

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
	"go.opentelemetry.io/collector/cmd/builder/internal/config"
)

const (
	skipCompilationFlag            = "skip-compilation"
	skipGetModulesFlag             = "skip-get-modules"
	distributionNameFlag           = "name"
	distributionDescriptionFlag    = "description"
	distributionVersionFlag        = "version"
	distributionOtelColVersionFlag = "otelcol-version"
	distributionOutputPathFlag     = "output-path"
	distributionGoFlag             = "go"
	distributionModuleFlag         = "module"
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
	cmd.Flags().BoolVar(&cfg.SkipCompilation, skipCompilationFlag, false, "Whether builder should only generate go code with no compile of the collector (default false)")
	cmd.Flags().BoolVar(&cfg.SkipGetModules, skipGetModulesFlag, false, "Whether builder should skip updating go.mod and retrieve Go module list (default false)")
	cmd.Flags().StringVar(&cfg.Distribution.Name, distributionNameFlag, "otelcol-custom", "The executable name for the OpenTelemetry Collector distribution")
	if err := cmd.Flags().MarkDeprecated(distributionNameFlag, "use config distribution::name"); err != nil {
		return nil, err
	}
	cmd.Flags().StringVar(&cfg.Distribution.Description, distributionDescriptionFlag, "Custom OpenTelemetry Collector distribution", "A descriptive name for the OpenTelemetry Collector distribution")
	if err := cmd.Flags().MarkDeprecated(distributionDescriptionFlag, "use config distribution::description"); err != nil {
		return nil, err
	}
	cmd.Flags().StringVar(&cfg.Distribution.Version, distributionVersionFlag, "1.0.0", "The version for the OpenTelemetry Collector distribution")
	if err := cmd.Flags().MarkDeprecated(distributionVersionFlag, "use config distribution::version"); err != nil {
		return nil, err
	}
	cmd.Flags().StringVar(&cfg.Distribution.OtelColVersion, distributionOtelColVersionFlag, cfg.Distribution.OtelColVersion, "Which version of OpenTelemetry Collector to use as base")
	if err := cmd.Flags().MarkDeprecated(distributionOtelColVersionFlag, "use config distribution::otelcol_version"); err != nil {
		return nil, err
	}
	cmd.Flags().StringVar(&cfg.Distribution.OutputPath, distributionOutputPathFlag, cfg.Distribution.OutputPath, "Where to write the resulting files")
	if err := cmd.Flags().MarkDeprecated(distributionOutputPathFlag, "use config distribution::output_path"); err != nil {
		return nil, err
	}
	cmd.Flags().StringVar(&cfg.Distribution.Go, distributionGoFlag, "", "The Go binary to use during the compilation phase. Default: go from the PATH")
	if err := cmd.Flags().MarkDeprecated(distributionGoFlag, "use config distribution::go"); err != nil {
		return nil, err
	}
	cmd.Flags().StringVar(&cfg.Distribution.Module, distributionModuleFlag, "go.opentelemetry.io/collector/cmd/builder", "The Go module for the new distribution")
	if err := cmd.Flags().MarkDeprecated(distributionModuleFlag, "use config distribution::module"); err != nil {
		return nil, err
	}

	// version of this binary
	cmd.AddCommand(versionCommand())

	return cmd, nil
}

func initConfig(flags *flag.FlagSet) error {
	cfg.Logger.Info("OpenTelemetry Collector Builder",
		zap.String("version", version), zap.String("date", date))

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
		return strings.ReplaceAll(s, ".", "_")
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
	cfg.Replaces = cfgFromFile.Replaces
	cfg.Excludes = cfgFromFile.Excludes

	if !flags.Changed(skipCompilationFlag) && cfgFromFile.SkipCompilation {
		cfg.SkipCompilation = cfgFromFile.SkipCompilation
	}
	if !flags.Changed(skipGetModulesFlag) && cfgFromFile.SkipGetModules {
		cfg.SkipGetModules = cfgFromFile.SkipGetModules
	}
	if !flags.Changed(distributionNameFlag) && cfgFromFile.Distribution.Name != "" {
		cfg.Distribution.Name = cfgFromFile.Distribution.Name
	}
	if !flags.Changed(distributionDescriptionFlag) && cfgFromFile.Distribution.Description != "" {
		cfg.Distribution.Description = cfgFromFile.Distribution.Description
	}
	if !flags.Changed(distributionVersionFlag) && cfgFromFile.Distribution.Version != "" {
		cfg.Distribution.Version = cfgFromFile.Distribution.Version
	}
	if !flags.Changed(distributionOtelColVersionFlag) && cfgFromFile.Distribution.OtelColVersion != "" {
		cfg.Distribution.OtelColVersion = cfgFromFile.Distribution.OtelColVersion
	}
	if !flags.Changed(distributionOutputPathFlag) && cfgFromFile.Distribution.OutputPath != "" {
		cfg.Distribution.OutputPath = cfgFromFile.Distribution.OutputPath
	}
	if !flags.Changed(distributionGoFlag) && cfgFromFile.Distribution.Go != "" {
		cfg.Distribution.Go = cfgFromFile.Distribution.Go
	}
	if !flags.Changed(distributionModuleFlag) && cfgFromFile.Distribution.Module != "" {
		cfg.Distribution.Module = cfgFromFile.Distribution.Module
	}
}
