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
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
)

var (
	cfgFile string
	cfg     = builder.NewDefaultConfig()
	k       = koanf.New(".")
)

// Command is the main entrypoint for this application
func Command() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use: "ocb",
		Long: fmt.Sprintf("OpenTelemetry Collector Builder (%s)", version) + `

ocb generates a custom OpenTelemetry Collector binary using the
build configuration given by the "--config" argument.
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(); err != nil {
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

	cmd.Flags().StringVar(&cfgFile, "config", "", "build configuration file")

	// A build configuration file is always required, and there's no
	// default. We can relax this in future by embedding the default
	// config that is used to build otelcorecol.
	if err := cmd.MarkFlagRequired("config"); err != nil {
		panic(err) // Only fails if the usage message is empty, which is a programmer error.
	}

	// the distribution parameters, which we accept as CLI flags as well
	cmd.Flags().BoolVar(&cfg.SkipCompilation, "skip-compilation", false, "Whether builder should only generate go code with no compile of the collector (default false)")
	cmd.Flags().StringVar(&cfg.Distribution.Name, "name", "otelcol-custom", "The executable name for the OpenTelemetry Collector distribution")
	cmd.Flags().StringVar(&cfg.Distribution.Description, "description", "Custom OpenTelemetry Collector distribution", "A descriptive name for the OpenTelemetry Collector distribution")
	cmd.Flags().StringVar(&cfg.Distribution.Version, "version", "1.0.0", "The version for the OpenTelemetry Collector distribution")
	cmd.Flags().StringVar(&cfg.Distribution.OtelColVersion, "otelcol-version", cfg.Distribution.OtelColVersion, "Which version of OpenTelemetry Collector to use as base")
	cmd.Flags().StringVar(&cfg.Distribution.OutputPath, "output-path", cfg.Distribution.OutputPath, "Where to write the resulting files")
	cmd.Flags().StringVar(&cfg.Distribution.Go, "go", "", "The Go binary to use during the compilation phase. Default: go from the PATH")
	cmd.Flags().StringVar(&cfg.Distribution.Module, "module", "go.opentelemetry.io/collector/cmd/builder", "The Go module for the new distribution")

	// version of this binary
	cmd.AddCommand(versionCommand())

	if err := k.Load(posflag.Provider(cmd.Flags(), ".", k), nil); err != nil {
		cfg.Logger.Error("failed to load command line arguments", zap.Error(err))
	}

	return cmd, nil
}

func initConfig() error {
	cfg.Logger.Info("OpenTelemetry Collector Builder", zap.String("version", version), zap.String("date", date))

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

	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		cfg.Logger.Error("failed to unmarshal config", zap.Error(err))
		return err
	}

	cfg.Logger.Info("Using config file", zap.String("path", cfgFile))
	return nil
}
