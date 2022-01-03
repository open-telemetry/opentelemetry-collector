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

package builder // import "go.opentelemetry.io/collector/cmd/builder/internal/builder"

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

const defaultOtelColVersion = "0.41.0"

// ErrInvalidGoMod indicates an invalid gomod
var ErrInvalidGoMod = errors.New("invalid gomod specification for module")

// ErrDeprecatedCore indicates deprecated core
var ErrDeprecatedCore = errors.New("mod.Core has deprecated, you should not be used anymore and required to be set mod.GoMod instead")

// Config holds the builder's configuration
type Config struct {
	Logger          *zap.Logger
	SkipCompilation bool

	Distribution Distribution `mapstructure:"dist"`
	Exporters    []Module     `mapstructure:"exporters"`
	Extensions   []Module     `mapstructure:"extensions"`
	Receivers    []Module     `mapstructure:"receivers"`
	Processors   []Module     `mapstructure:"processors"`
	Replaces     []string     `mapstructure:"replaces"`
	Excludes     []string     `mapstructure:"excludes"`
}

// Distribution holds the parameters for the final binary
type Distribution struct {
	Module         string `mapstructure:"module"`
	Name           string `mapstructure:"name"`
	Go             string `mapstructure:"go"`
	Description    string `mapstructure:"description"`
	OtelColVersion string `mapstructure:"otelcol_version"`
	OutputPath     string `mapstructure:"output_path"`
	Version        string `mapstructure:"version"`
}

// Module represents a receiver, exporter, processor or extension for the distribution
type Module struct {
	Name   string `mapstructure:"name"`   // if not specified, this is package part of the go mod (last part of the path)
	Import string `mapstructure:"import"` // if not specified, this is the path part of the go mods
	GoMod  string `mapstructure:"gomod"`  // a gomod-compatible spec for the module
	Path   string `mapstructure:"path"`   // an optional path to the local version of this module
	Core   *bool  `mapstructure:"core"`   // whether this module comes from core. For this property isn't referred from anywhere, it might be removed. please see #15.
}

// DefaultConfig creates a new config, with default values
func DefaultConfig() Config {
	log, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to obtain a logger instance: %v", err))
	}

	outputDir, err := ioutil.TempDir("", "otelcol-distribution")
	if err != nil {
		log.Error("failed to obtain a temporary directory", zap.Error(err))
	}

	return Config{
		Logger: log,
		Distribution: Distribution{
			OutputPath:     outputDir,
			OtelColVersion: defaultOtelColVersion,
			Module:         "go.opentelemetry.io/collector/cmd/builder",
		},
	}
}

// Validate checks whether the current configuration is valid
func (c *Config) Validate() error {
	// #nosec G204
	if _, err := exec.Command(c.Distribution.Go, "env").CombinedOutput(); err != nil {
		path, err := exec.LookPath("go")
		if err != nil {
			return ErrGoNotFound
		}
		c.Distribution.Go = path
	}

	c.Logger.Info("Using go", zap.String("go-executable", c.Distribution.Go))

	return nil
}

// ParseModules will parse the Modules entries and populate the missing values
func (c *Config) ParseModules() error {
	var err error

	c.Extensions, err = parseModules(c.Extensions)
	if err != nil {
		return err
	}

	c.Receivers, err = parseModules(c.Receivers)
	if err != nil {
		return err
	}

	c.Exporters, err = parseModules(c.Exporters)
	if err != nil {
		return err
	}

	c.Processors, err = parseModules(c.Processors)
	if err != nil {
		return err
	}

	return nil
}

func parseModules(mods []Module) ([]Module, error) {
	var parsedModules []Module
	for _, mod := range mods {
		if mod.Core != nil {
			return mods, ErrDeprecatedCore
		}
		if mod.GoMod == "" {
			return mods, fmt.Errorf("%w, module: %q", ErrInvalidGoMod, mod.GoMod)
		}

		if mod.Import == "" {
			mod.Import = strings.Split(mod.GoMod, " ")[0]
		}

		if mod.Name == "" {
			parts := strings.Split(mod.Import, "/")
			mod.Name = parts[len(parts)-1]
		}

		if strings.HasPrefix(mod.Path, "./") {
			path, err := os.Getwd()
			if err != nil {
				return mods, fmt.Errorf("module has a relative Path element, but we couldn't get the current working dir: %v", err)
			}

			mod.Path = fmt.Sprintf("%s/%s", path, mod.Path[2:])
		}

		parsedModules = append(parsedModules, mod)
	}

	return parsedModules, nil
}
