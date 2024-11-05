// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder // import "go.opentelemetry.io/collector/cmd/builder/internal/builder"

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

const defaultBetaOtelColVersion = "v0.113.0"
const defaultStableOtelColVersion = "v1.17.0"

var (
	// errMissingGoMod indicates an empty gomod field
	errMissingGoMod = errors.New("missing gomod specification for module")
)

// Config holds the builder's configuration
type Config struct {
	Logger *zap.Logger

	OtelColVersion       string `mapstructure:"-"` // only used be the go.mod template
	SkipGenerate         bool   `mapstructure:"-"`
	SkipCompilation      bool   `mapstructure:"-"`
	SkipGetModules       bool   `mapstructure:"-"`
	SkipStrictVersioning bool   `mapstructure:"-"`
	LDFlags              string `mapstructure:"-"`
	Verbose              bool   `mapstructure:"-"`

	Distribution Distribution `mapstructure:"dist"`
	Exporters    []Module     `mapstructure:"exporters"`
	Extensions   []Module     `mapstructure:"extensions"`
	Receivers    []Module     `mapstructure:"receivers"`
	Processors   []Module     `mapstructure:"processors"`
	Connectors   []Module     `mapstructure:"connectors"`
	Providers    []Module     `mapstructure:"providers"`
	Converters   []Module     `mapstructure:"converters"`
	Replaces     []string     `mapstructure:"replaces"`
	Excludes     []string     `mapstructure:"excludes"`

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
	Module      string `mapstructure:"module"`
	Name        string `mapstructure:"name"`
	Go          string `mapstructure:"go"`
	Description string `mapstructure:"description"`
	// Deprecated: [v0.113.0] only here to return a detailed error and not failing during unmarshalling.
	OtelColVersion   string `mapstructure:"otelcol_version"`
	OutputPath       string `mapstructure:"output_path"`
	Version          string `mapstructure:"version"`
	BuildTags        string `mapstructure:"build_tags"`
	DebugCompilation bool   `mapstructure:"debug_compilation"`
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
		Providers: []Module{
			{GoMod: "go.opentelemetry.io/collector/confmap/provider/envprovider " + defaultStableOtelColVersion},
			{GoMod: "go.opentelemetry.io/collector/confmap/provider/fileprovider " + defaultStableOtelColVersion},
			{GoMod: "go.opentelemetry.io/collector/confmap/provider/httpprovider " + defaultStableOtelColVersion},
			{GoMod: "go.opentelemetry.io/collector/confmap/provider/httpsprovider " + defaultStableOtelColVersion},
			{GoMod: "go.opentelemetry.io/collector/confmap/provider/yamlprovider " + defaultStableOtelColVersion},
		},
	}, nil
}

// Validate checks whether the current configuration is valid
func (c *Config) Validate() error {
	if c.Distribution.OtelColVersion != "" {
		return errors.New("`otelcol_version` has been removed. To build with an older Collector API, use an older (aligned) builder version instead")
	}
	return errors.Join(
		validateModules("connector", c.Connectors),
		validateModules("converter", c.Converters),
		validateModules("exporter", c.Exporters),
		validateModules("extension", c.Extensions),
		validateModules("processor", c.Processors),
		validateModules("provider", c.Providers),
		validateModules("receiver", c.Receivers),
	)
}

// SetGoPath sets go path
func (c *Config) SetGoPath() error {
	if !c.SkipCompilation || !c.SkipGetModules {
		// #nosec G204
		if _, err := exec.Command(c.Distribution.Go, "env").CombinedOutput(); err != nil { // nolint G204
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
	return errors.Join(
		parseModules(c.Connectors),
		parseModules(c.Converters),
		parseModules(c.Exporters),
		parseModules(c.Extensions),
		parseModules(c.Processors),
		parseModules(c.Providers),
		parseModules(c.Receivers),
	)
}

func validateModules(name string, mods []Module) error {
	for i, mod := range mods {
		if mod.GoMod == "" {
			return fmt.Errorf("%s module at index %v: %w", name, i, errMissingGoMod)
		}
	}
	return nil
}

func parseModules(mods []Module) error {
	for i := range mods {
		if mods[i].Import == "" {
			mods[i].Import = strings.Split(mods[i].GoMod, " ")[0]
		}

		if mods[i].Name == "" {
			parts := strings.Split(mods[i].Import, "/")
			mods[i].Name = parts[len(parts)-1]
		}

		// Check if path is empty, otherwise filepath.Abs replaces it with current path ".".
		if mods[i].Path != "" {
			var err error
			mods[i].Path, err = filepath.Abs(mods[i].Path)
			if err != nil {
				return fmt.Errorf("module has a relative \"path\" element, but we couldn't resolve the current working dir: %w", err)
			}
			// Check if the path exists
			if _, err := os.Stat(mods[i].Path); os.IsNotExist(err) {
				return fmt.Errorf("filepath does not exist: %s", mods[i].Path)
			}
		}
	}

	return nil
}
