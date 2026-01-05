// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder // import "go.opentelemetry.io/collector/cmd/builder/internal/builder"

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	defaultBetaOtelColVersion   = "v0.142.0"
	defaultStableOtelColVersion = "v1.48.0"
)

// errMissingGoMod indicates an empty gomod field
var errMissingGoMod = errors.New("missing gomod specification for module")

// Config holds the builder's configuration
type Config struct {
	Logger *zap.Logger

	OtelColVersion       string `mapstructure:"-"` // only used be the go.mod template
	SkipGenerate         bool   `mapstructure:"-"`
	SkipCompilation      bool   `mapstructure:"-"`
	SkipGetModules       bool   `mapstructure:"-"`
	SkipStrictVersioning bool   `mapstructure:"-"`
	LDFlags              string `mapstructure:"-"`
	LDSet                bool   `mapstructure:"-"` // only used to override LDFlags
	GCFlags              string `mapstructure:"-"`
	GCSet                bool   `mapstructure:"-"` // only used to override GCFlags
	Verbose              bool   `mapstructure:"-"`

	Distribution      Distribution `mapstructure:"dist"`
	Exporters         []Module     `mapstructure:"exporters"`
	Extensions        []Module     `mapstructure:"extensions"`
	Receivers         []Module     `mapstructure:"receivers"`
	Processors        []Module     `mapstructure:"processors"`
	Connectors        []Module     `mapstructure:"connectors"`
	ConfmapProviders  []Module     `mapstructure:"providers"`
	ConfmapConverters []Module     `mapstructure:"converters"`
	Replaces          []string     `mapstructure:"replaces"`
	Excludes          []string     `mapstructure:"excludes"`

	ConfResolver ConfResolver `mapstructure:"conf_resolver"`

	downloadModules retry `mapstructure:"-"`
}

type ConfResolver struct {
	// When set, will be used to set the CollectorSettings.ConfResolver.DefaultScheme value,
	// which determines how the Collector interprets URIs that have no scheme, such as ${ENV}.
	// See https://pkg.go.dev/go.opentelemetry.io/collector/confmap#ResolverSettings for more details.
	DefaultURIScheme string `mapstructure:"default_uri_scheme"`
}

// Distribution holds the parameters for the final binary
type Distribution struct {
	Module           string `mapstructure:"module"`
	Name             string `mapstructure:"name"`
	Go               string `mapstructure:"go"`
	Description      string `mapstructure:"description"`
	OutputPath       string `mapstructure:"output_path"`
	Version          string `mapstructure:"version"`
	BuildTags        string `mapstructure:"build_tags"`
	DebugCompilation bool   `mapstructure:"debug_compilation"`
	CGoEnabled       bool   `mapstructure:"cgo_enabled"`
}

// Module represents a receiver, exporter, processor or extension for the distribution
type Module struct {
	Name   string `mapstructure:"name"`   // if not specified, this is package part of the go mod (last part of the path)
	Import string `mapstructure:"import"` // if not specified, this is the path part of the go mods
	GoMod  string `mapstructure:"gomod"`  // a gomod-compatible spec for the module
	Path   string `mapstructure:"path"`   // an optional path to the local version of this module
}

type retry struct {
	numRetries int
	wait       time.Duration
}

// NewDefaultConfig creates a new config, with default values
func NewDefaultConfig() (*Config, error) {
	log, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to obtain a logger instance: %v", err))
	}

	outputDir, err := os.MkdirTemp("", "otelcol-distribution")
	if err != nil {
		return nil, err
	}

	return &Config{
		OtelColVersion: defaultBetaOtelColVersion,
		Logger:         log,
		Distribution: Distribution{
			OutputPath: outputDir,
			Module:     "go.opentelemetry.io/collector/cmd/builder",
		},
		// basic retry if error from go mod command (in case of transient network error).
		// retry 3 times with 5 second spacing interval
		downloadModules: retry{
			numRetries: 3,
			wait:       5 * time.Second,
		},
		ConfmapProviders: []Module{
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/envprovider " + defaultStableOtelColVersion,
			},
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/fileprovider " + defaultStableOtelColVersion,
			},
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/httpprovider " + defaultStableOtelColVersion,
			},
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/httpsprovider " + defaultStableOtelColVersion,
			},
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/yamlprovider " + defaultStableOtelColVersion,
			},
		},
	}, nil
}

// Validate checks whether the current configuration is valid
func (c *Config) Validate() error {
	return multierr.Combine(
		validateModules("extension", c.Extensions),
		validateModules("receiver", c.Receivers),
		validateModules("exporter", c.Exporters),
		validateModules("processor", c.Processors),
		validateModules("connector", c.Connectors),
		validateModules("provider", c.ConfmapProviders),
		validateModules("converter", c.ConfmapConverters),
	)
}

// SetGoPath sets go path
func (c *Config) SetGoPath() error {
	if !c.SkipCompilation || !c.SkipGetModules {
		//nolint:gosec // #nosec G204
		if _, err := exec.Command(c.Distribution.Go, "env").CombinedOutput(); err != nil {
			path, err := exec.LookPath("go")
			if err != nil {
				return ErrGoNotFound
			}
			c.Distribution.Go = path
		}
		c.Logger.Info("Using go", zap.String("go-executable", c.Distribution.Go))
	}
	return nil
}

// ParseModules will parse the Modules entries and populate the missing values
func (c *Config) ParseModules() error {
	var err error
	usedNames := make(map[string]int)

	c.Extensions, err = parseModules(c.Extensions, usedNames)
	if err != nil {
		return err
	}

	c.Receivers, err = parseModules(c.Receivers, usedNames)
	if err != nil {
		return err
	}

	c.Exporters, err = parseModules(c.Exporters, usedNames)
	if err != nil {
		return err
	}

	c.Processors, err = parseModules(c.Processors, usedNames)
	if err != nil {
		return err
	}

	c.Connectors, err = parseModules(c.Connectors, usedNames)
	if err != nil {
		return err
	}

	c.ConfmapProviders, err = parseModules(c.ConfmapProviders, usedNames)
	if err != nil {
		return err
	}
	c.ConfmapConverters, err = parseModules(c.ConfmapConverters, usedNames)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) allComponents() []Module {
	return slices.Concat[[]Module](c.Exporters, c.Receivers, c.Processors, c.Extensions, c.Connectors, c.ConfmapProviders, c.ConfmapConverters)
}

func validateModules(name string, mods []Module) error {
	for i, mod := range mods {
		if mod.GoMod == "" {
			return fmt.Errorf("%s module at index %v: %w", name, i, errMissingGoMod)
		}
	}
	return nil
}

func parseModules(mods []Module, usedNames map[string]int) ([]Module, error) {
	var parsedModules []Module
	for _, mod := range mods {
		if mod.Import == "" {
			mod.Import = strings.Split(mod.GoMod, " ")[0]
		}

		if mod.Name == "" {
			parts := strings.Split(mod.Import, "/")
			mod.Name = parts[len(parts)-1]
		}

		originalModName := mod.Name
		if count, exists := usedNames[mod.Name]; exists {
			var newName string
			for {
				newName = fmt.Sprintf("%s%d", mod.Name, count+1)
				if _, transformedExists := usedNames[newName]; !transformedExists {
					break
				}
				count++
			}
			mod.Name = newName
			usedNames[newName] = 1
		}
		usedNames[originalModName] = 1

		// Check if path is empty, otherwise filepath.Abs replaces it with current path ".".
		if mod.Path != "" {
			var err error
			mod.Path, err = filepath.Abs(mod.Path)
			if err != nil {
				return mods, fmt.Errorf("module has a relative \"path\" element, but we couldn't resolve the current working dir: %w", err)
			}
			// Check if the path exists
			if _, err := os.Stat(mod.Path); os.IsNotExist(err) {
				return mods, fmt.Errorf("filepath does not exist: %s", mod.Path)
			}
		}

		parsedModules = append(parsedModules, mod)
	}

	return parsedModules, nil
}
