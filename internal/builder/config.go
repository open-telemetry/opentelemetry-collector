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

package builder

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

const defaultOtelColVersion = "0.26.0"

// ErrInvalidGoMod indicates an invalid gomod
var ErrInvalidGoMod = errors.New("invalid gomod specification for module")

// Config holds the builder's configuration
type Config struct {
	Logger logr.Logger

	Distribution Distribution `mapstructure:"dist"`
	Exporters    []Module     `mapstructure:"exporters"`
	Extensions   []Module     `mapstructure:"extensions"`
	Receivers    []Module     `mapstructure:"receivers"`
	Processors   []Module     `mapstructure:"processors"`
	Replaces     []string     `mapstructure:"replaces"`
}

// Distribution holds the parameters for the final binary
type Distribution struct {
	Module         string `mapstructure:"module"`
	ExeName        string `mapstructure:"name"`
	Go             string `mapstructure:"go"`
	LongName       string `mapstructure:"description"`
	OtelColVersion string `mapstructure:"otelcol_version"`
	IncludeCore    bool   `mapstructure:"include_core"`
	OutputPath     string `mapstructure:"output_path"`
	Version        string `mapstructure:"version"`
}

// Module represents a receiver, exporter, processor or extension for the distribution
type Module struct {
	Name   string `mapstructure:"name"`   // if not specified, this is package part of the go mod (last part of the path)
	Import string `mapstructure:"import"` // if not specified, this is the path part of the go mods
	GoMod  string `mapstructure:"gomod"`  // a gomod-compatible spec for the module
	Path   string `mapstructure:"path"`   // an optional path to the local version of this module
	Core   bool   `mapstructure:"core"`   // whether this module comes from core, meaning that no further dependencies will be added
}

// DefaultConfig creates a new config, with default values
func DefaultConfig() Config {
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to obtain a logger instance: %v", err))
	}
	log := zapr.NewLogger(zapLog)

	outputDir, err := ioutil.TempDir("", "otelcol-distribution")
	if err != nil {
		log.Error(err, "failed to obtain a temporary directory")
	}

	return Config{
		Logger: log,
		Distribution: Distribution{
			OutputPath:     outputDir,
			OtelColVersion: defaultOtelColVersion,
		},
	}
}

// Validate checks whether the current configuration is valid
func (c Config) Validate() error {
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
	parsedModules := []Module{}
	for _, mod := range mods {
		if len(mod.GoMod) == 0 && !mod.Core {
			return mods, fmt.Errorf("%w, module: %q", ErrInvalidGoMod, mod.GoMod)
		}

		if len(mod.Import) == 0 {
			mod.Import = strings.Split(mod.GoMod, " ")[0]
		}

		if len(mod.Name) == 0 {
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
