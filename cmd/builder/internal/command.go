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
	"github.com/knadh/koanf/providers/posflag"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
)

const SkipComplicationFlag = "skip-compilation"
const DistributionNameFlag = "name"
const DistributionDescriptionFlag = "description"
const DistributionVersionFlag = "version"
const DistributionOtelColVersionFlag = "otelcol-version"
const DistributionOutputPathFlag = "output-path"
const DistributionGoFlag = "go"
const DistributionModuleFlag = "module"

var (
	cfgFile string
	cfg     = builder.NewDefaultConfig()
	k       = koanf.New(".")
)

// Command is the main entrypoint for this application
func Command() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:  "builder",
		Long: fmt.Sprintf("OpenTelemetry Collector distribution builder (%s)", version),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(cmd.Flags()); err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				cfg.Logger.Error("invalid configuration", zap.Error(err))
				return err
			}

			if err := cfg.ParseModules(); err != nil {
				cfg.Logger.Error("invalid module configuration", zap.Error(err))
				return err
			}

			return builder.GenerateAndCompile(cfg)
		},
	}

	// the external config file
	cmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.otelcol-builder.yaml)")

	// the distribution parameters, which we accept as CLI flags as well
	cmd.Flags().BoolVar(&cfg.SkipCompilation, SkipComplicationFlag, false, "Whether builder should only generate go code with no compile of the collector (default false)")
	cmd.Flags().StringVar(&cfg.Distribution.Name, DistributionNameFlag, "otelcol-custom", "The executable name for the OpenTelemetry Collector distribution")
	cmd.Flags().StringVar(&cfg.Distribution.Description, DistributionDescriptionFlag, "Custom OpenTelemetry Collector distribution", "A descriptive name for the OpenTelemetry Collector distribution")
	cmd.Flags().StringVar(&cfg.Distribution.Version, DistributionVersionFlag, "1.0.0", "The version for the OpenTelemetry Collector distribution")
	cmd.Flags().StringVar(&cfg.Distribution.OtelColVersion, DistributionOtelColVersionFlag, cfg.Distribution.OtelColVersion, "Which version of OpenTelemetry Collector to use as base")
	cmd.Flags().StringVar(&cfg.Distribution.OutputPath, DistributionOutputPathFlag, cfg.Distribution.OutputPath, "Where to write the resulting files")
	cmd.Flags().StringVar(&cfg.Distribution.Go, DistributionGoFlag, "", "The Go binary to use during the compilation phase. Default: go from the PATH")
	cmd.Flags().StringVar(&cfg.Distribution.Module, DistributionModuleFlag, "go.opentelemetry.io/collector/cmd/builder", "The Go module for the new distribution")

	// version of this binary
	cmd.AddCommand(versionCommand())

	if err := k.Load(posflag.Provider(cmd.Flags(), ".", k), nil); err != nil {
		cfg.Logger.Error("failed to load command line arguments", zap.Error(err))
	}

	return cmd, nil
}

func initConfig(flags *flag.FlagSet) error {
	cfg.Logger.Info("OpenTelemetry Collector distribution builder", zap.String(DistributionVersionFlag, version), zap.String("date", date))
	// use the default path if there is no config file being specified
	if cfgFile == "" {
		cfgFile = "$HOME/.otelcol-builder.yaml"
	}

	// load the config file
	if err := k.Load(file.Provider(cfgFile), yaml.Parser()); err != nil {
		cfg.Logger.Error("failed to load config file", zap.String("config-file", cfgFile), zap.Error(err))
	}

	// handle env variables
	if err := k.Load(env.Provider("", ".", func(s string) string {
		return strings.ReplaceAll(s, ".", "_")
	}), nil); err != nil {
		cfg.Logger.Error("failed to load env var", zap.Error(err))
	}

	cfgFromFile := builder.Config{}
	if err := k.UnmarshalWithConf("", &cfgFromFile, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		cfg.Logger.Error("failed to unmarshal config", zap.Error(err))
		return err
	}

	cfg.Exporters = cfgFromFile.Exporters
	cfg.Extensions = cfgFromFile.Extensions
	cfg.Receivers = cfgFromFile.Receivers
	cfg.Processors = cfgFromFile.Processors
	cfg.Replaces = cfgFromFile.Replaces
	cfg.Excludes = cfgFromFile.Excludes

	if !flags.Changed(SkipComplicationFlag) && cfgFromFile.SkipCompilation {
		cfg.SkipCompilation = cfgFromFile.SkipCompilation
	}
	if !flags.Changed(DistributionNameFlag) && cfgFromFile.Distribution.Name != "" {
		cfg.Distribution.Name = cfgFromFile.Distribution.Name
	}
	if !flags.Changed(DistributionDescriptionFlag) && cfgFromFile.Distribution.Description != "" {
		cfg.Distribution.Description = cfgFromFile.Distribution.Description
	}
	if !flags.Changed(DistributionVersionFlag) && cfgFromFile.Distribution.Version != "" {
		cfg.Distribution.Version = cfgFromFile.Distribution.Version
	}
	if !flags.Changed(DistributionOtelColVersionFlag) && cfgFromFile.Distribution.OtelColVersion != "" {
		cfg.Distribution.OtelColVersion = cfgFromFile.Distribution.OtelColVersion
	}
	if !flags.Changed(DistributionOutputPathFlag) && cfgFromFile.Distribution.OutputPath != "" {
		cfg.Distribution.OutputPath = cfgFromFile.Distribution.OutputPath
	}
	if !flags.Changed(DistributionGoFlag) && cfgFromFile.Distribution.Go != "" {
		cfg.Distribution.Go = cfgFromFile.Distribution.Go
	}
	if !flags.Changed(DistributionModuleFlag) && cfgFromFile.Distribution.Module != "" {
		cfg.Distribution.Module = cfgFromFile.Distribution.Module
	}

	cfg.Logger.Info("Using config file", zap.String("path", cfgFile))
	return nil
}
