// Copyright 2020 OpenTelemetry Authors
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

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-telemetry/opentelemetry-collector-builder/internal/builder"
)

var (
	version = "dev"
	date    = "unknown"

	cfgFile string
	cfg     = builder.DefaultConfig()

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Version of opentelemetry-collector-builder",
		Long:  "Prints the version of opentelemetry-collector-builder binary",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
)

// Execute is the main entrypoint for this application
func Execute() error {
	cobra.OnInitialize(initConfig)

	cmd := &cobra.Command{
		Use:  "opentelemetry-collector-builder",
		Long: fmt.Sprintf("OpenTelemetry Collector distribution builder (%s)", version),
		RunE: func(cmd *cobra.Command, args []string) error {

			if err := cfg.Validate(); err != nil {
				cfg.Logger.Error(err, "invalid configuration")
				return err
			}

			if err := cfg.ParseModules(); err != nil {
				cfg.Logger.Error(err, "invalid module configuration")
				return err
			}

			return builder.GenerateAndCompile(cfg)
		},
	}

	// the external config file
	cmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.otelcol-builder.yaml)")

	// the distribution parameters, which we accept as CLI flags as well
	cmd.Flags().BoolVar(&cfg.SkipCompilation, "skip-compilation", false, "Whether builder should only generate go code with no compile of the collector (default false)")
	cmd.Flags().StringVar(&cfg.Distribution.ExeName, "name", "otelcol-custom", "The executable name for the OpenTelemetry Collector distribution")
	cmd.Flags().StringVar(&cfg.Distribution.LongName, "description", "Custom OpenTelemetry Collector distribution", "A descriptive name for the OpenTelemetry Collector distribution")
	cmd.Flags().StringVar(&cfg.Distribution.Version, "version", "1.0.0", "The version for the OpenTelemetry Collector distribution")
	cmd.Flags().BoolVar(&cfg.Distribution.IncludeCore, "include-core", true, "Whether the core components should be included in the distribution")
	cmd.Flags().StringVar(&cfg.Distribution.OtelColVersion, "otelcol-version", cfg.Distribution.OtelColVersion, "Which version of OpenTelemetry Collector to use as base")
	cmd.Flags().StringVar(&cfg.Distribution.OutputPath, "output-path", cfg.Distribution.OutputPath, "Where to write the resulting files")
	cmd.Flags().StringVar(&cfg.Distribution.Go, "go", "", "The Go binary to use during the compilation phase. Default: go from the PATH")
	cmd.Flags().StringVar(&cfg.Distribution.Module, "module", "github.com/open-telemetry/opentelemetry-collector-builder", "The Go module for the new distribution")

	// version of this binary
	cmd.AddCommand(versionCmd)

	// tie Viper to flags
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		cfg.Logger.Error(err, "failed to bind flags")
		return err
	}

	if err := cmd.Execute(); err != nil {
		cfg.Logger.Error(err, "failed to run")
		return err
	}

	return nil
}

func initConfig() {
	cfg.Logger.Info("OpenTelemetry Collector distribution builder", "version", version, "date", date)

	// a couple of Viper goodies, to make it easier to use env vars when flags are not desirable
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// load values from config file -- required for the modules configuration
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("$HOME")
		viper.SetConfigName(".otelcol-builder")
	}

	// load the config file
	if err := viper.ReadInConfig(); err == nil {
		cfg.Logger.Info("Using config file", "path", viper.ConfigFileUsed())
	}

	// convert Viper's internal state into our configuration object
	if err := viper.Unmarshal(&cfg); err != nil {
		cfg.Logger.Error(err, "failed to parse the config")
		cobra.CheckErr(err)
		return
	}
}
