// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder // import "go.opentelemetry.io/collector/cmd/builder/internal/builder"

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const defaultOtelColVersion = "0.111.0"

var goVersionPackageURL = "https://golang.org/dl/"

// ErrMissingGoMod indicates an empty gomod field
var ErrMissingGoMod = errors.New("missing gomod specification for module")

var readBuildInfo = debug.ReadBuildInfo

// Config holds the builder's configuration
type Config struct {
	Logger *zap.Logger

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
	Providers    *[]Module    `mapstructure:"providers"`
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
	Module                   string `mapstructure:"module"`
	Name                     string `mapstructure:"name"`
	Go                       string `mapstructure:"go"`
	Description              string `mapstructure:"description"`
	OtelColVersion           string `mapstructure:"otelcol_version"`
	RequireOtelColModule     bool   `mapstructure:"-"` // required for backwards-compatibility with builds older than 0.86.0
	SupportsConfmapFactories bool   `mapstructure:"-"` // Required for backwards-compatibility with builds older than 0.99.0
	SupportsComponentModules bool   `mapstructure:"-"` // Required for backwards-compatibility with builds older than 0.106.0
	OutputPath               string `mapstructure:"output_path"`
	Version                  string `mapstructure:"version"`
	BuildTags                string `mapstructure:"build_tags"`
	DebugCompilation         bool   `mapstructure:"debug_compilation"`
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
func NewDefaultConfig() Config {
	log, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to obtain a logger instance: %v", err))
	}

	outputDir, err := os.MkdirTemp("", "otelcol-distribution")
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
		// basic retry if error from go mod command (in case of transient network error).
		// retry 3 times with 5 second spacing interval
		downloadModules: retry{
			numRetries: 3,
			wait:       5 * time.Second,
		},
	}
}

// Validate checks whether the current configuration is valid
func (c *Config) Validate() error {
	var providersError error
	if c.Providers != nil {
		providersError = validateModules("provider", *c.Providers)
	}
	return multierr.Combine(
		validateModules("extension", c.Extensions),
		validateModules("receiver", c.Receivers),
		validateModules("exporter", c.Exporters),
		validateModules("processor", c.Processors),
		validateModules("connector", c.Connectors),
		providersError,
	)
}

func getGoVersionFromBuildInfo() (string, error) {
	info, ok := readBuildInfo()
	if !ok {
		return "", fmt.Errorf("failed to read Go build info")
	}
	version := strings.TrimPrefix(info.GoVersion, "go")
	return version, nil
}
func sanitizeExtractPath(filePath string, destination string) error {
	// to avoid zip slip (writing outside of the destination), we resolve
	// the target path, and make sure it's nested in the intended
	// destination, or bail otherwise.
	destpath := filepath.Join(destination, filePath)
	if !strings.HasPrefix(destpath, destination) {
		return fmt.Errorf("%s: illegal file path", filePath)
	}
	return nil
}

func downloadGoBinary(version string) error {
	platform := runtime.GOOS
	arch := runtime.GOARCH
	originalGoURL := goVersionPackageURL
	goVersionPackageURL = fmt.Sprintf(goVersionPackageURL+"/go%s.%s-%s.tar.gz", version, platform, arch)
	defer func() {
		goVersionPackageURL = originalGoURL
	}()
	client := http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	request, err := http.NewRequest(http.MethodGet, goVersionPackageURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Go binary: %s", resp.Status)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	if err := os.MkdirAll(filepath.Join(os.TempDir(), "go"), 0750); err != nil {
		return err
	}

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		err = sanitizeExtractPath(header.Name, os.TempDir())
		if err == nil {
			target := filepath.Join(os.TempDir(), header.Name)
			switch header.Typeflag {
			case tar.TypeDir:
				if err := os.MkdirAll(target, 0750); err != nil {
					return err
				}
			case tar.TypeReg:
				f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
				if err != nil {
					return err
				}
				// Copy up to 500MB of data to avoid gosec warning; current Go distributions are arount 250MB
				if _, err := io.CopyN(f, tr, 500000000); err != nil {
					f.Close()
					if !errors.Is(err, io.EOF) {
						return err
					}
				}
				f.Close()
			}

		}
	}

	return nil

}

func removeGoTempDir() error {
	goTempDir := filepath.Join(os.TempDir(), "go")
	var err error
	if _, err = os.Stat(goTempDir); err == nil {
		if err = os.RemoveAll(goTempDir); err != nil {
			return fmt.Errorf("failed to remove go temp directory: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check go temp directory: %w", err)
	}
	return nil
}

// SetGoPath sets go path, and if go binary not found on path, downloads it
func (c *Config) SetGoPath() error {
	if !c.SkipCompilation || !c.SkipGetModules {
		// #nosec G204
		if _, err := exec.Command(c.Distribution.Go, "env").CombinedOutput(); err != nil { // nolint G204
			path, err := exec.LookPath("go")
			if err != nil {
				if runtime.GOOS == "windows" {
					return fmt.Errorf("failed to find go executable in PATH, please install Go from %s", goVersionPackageURL)
				}
				c.Logger.Info("Failed to find go executable in PATH, downloading Go binary")
				goVersion, err := getGoVersionFromBuildInfo()
				if err != nil {
					return err
				}
				c.Logger.Info(fmt.Sprintf("Downloading Go version %s from "+filepath.Join(goVersionPackageURL, fmt.Sprintf("go%s.%s-%s.tar.gz", goVersion, runtime.GOOS, runtime.GOARCH)), goVersion))
				if err := downloadGoBinary(goVersion); err != nil {
					return err
				}
				path = filepath.Join(os.TempDir(), "go", "bin", "go")
				c.Logger.Info(fmt.Sprintf("Installed go at temporary path: %s", path))
			}
			c.Distribution.Go = path
		}
		c.Logger.Info("Using go", zap.String("go-executable", c.Distribution.Go))
	}
	return nil
}

func (c *Config) SetBackwardsCompatibility() error {
	// Get the version of the collector
	otelColVersion, err := version.NewVersion(c.Distribution.OtelColVersion)
	if err != nil {
		return err
	}

	// check whether we need to adjust the core API module import
	constraint, err := version.NewConstraint(">= 0.86.0")
	if err != nil {
		return err
	}

	c.Distribution.RequireOtelColModule = constraint.Check(otelColVersion)

	// check whether confmap factories are supported
	constraint, err = version.NewConstraint(">= 0.99.0")
	if err != nil {
		return err
	}

	c.Distribution.SupportsConfmapFactories = constraint.Check(otelColVersion)

	// check whether go modules are recorded for components
	constraint, err = version.NewConstraint(">= 0.106.0")
	if err != nil {
		return err
	}

	c.Distribution.SupportsComponentModules = constraint.Check(otelColVersion)

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

	c.Connectors, err = parseModules(c.Connectors)
	if err != nil {
		return err
	}

	if c.Providers != nil {
		providers, err := parseModules(*c.Providers)
		if err != nil {
			return err
		}
		c.Providers = &providers
	} else {
		providers, err := parseModules([]Module{
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/envprovider v" + c.Distribution.OtelColVersion,
			},
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/fileprovider v" + c.Distribution.OtelColVersion,
			},
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/httpprovider v" + c.Distribution.OtelColVersion,
			},
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/httpsprovider v" + c.Distribution.OtelColVersion,
			},
			{
				GoMod: "go.opentelemetry.io/collector/confmap/provider/yamlprovider v" + c.Distribution.OtelColVersion,
			},
		})
		if err != nil {
			return err
		}
		c.Providers = &providers
	}

	return nil
}

func validateModules(name string, mods []Module) error {
	for i, mod := range mods {
		if mod.GoMod == "" {
			return fmt.Errorf("%s module at index %v: %w", name, i, ErrMissingGoMod)
		}
	}
	return nil
}

func parseModules(mods []Module) ([]Module, error) {
	var parsedModules []Module
	for _, mod := range mods {
		if mod.Import == "" {
			mod.Import = strings.Split(mod.GoMod, " ")[0]
		}

		if mod.Name == "" {
			parts := strings.Split(mod.Import, "/")
			mod.Name = parts[len(parts)-1]
		}

		// Check if path is empty, otherwise filepath.Abs replaces it with current path ".".
		if mod.Path != "" {
			var err error
			mod.Path, err = filepath.Abs(mod.Path)
			if err != nil {
				return mods, fmt.Errorf("module has a relative \"path\" element, but we couldn't resolve the current working dir: %w", err)
			}
		}

		parsedModules = append(parsedModules, mod)
	}

	return parsedModules, nil
}
