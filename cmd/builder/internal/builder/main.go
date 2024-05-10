// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder // import "go.opentelemetry.io/collector/cmd/builder/internal/builder"

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"go.uber.org/zap"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

var (
	// ErrGoNotFound is returned when a Go binary hasn't been found
	ErrGoNotFound      = errors.New("go binary not found")
	ErrDepNotFound     = errors.New("dependency not found in go mod file")
	ErrVersionMismatch = errors.New("mismatch in go.mod and builder configuration versions")
	errDownloadFailed  = errors.New("failed to download go modules")
	errCompileFailed   = errors.New("failed to compile the OpenTelemetry Collector distribution")
	skipStrictMsg      = "Use --skip-strict-versioning to temporarily disable this check. This flag will be removed in a future minor version"
)

func runGoCommand(cfg Config, args ...string) ([]byte, error) {
	if cfg.Verbose {
		cfg.Logger.Info("Running go subcommand.", zap.Any("arguments", args))
	}

	// #nosec G204 -- cfg.Distribution.Go is trusted to be a safe path and the caller is assumed to have carried out necessary input validation
	cmd := exec.Command(cfg.Distribution.Go, args...)
	cmd.Dir = cfg.Distribution.OutputPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go subcommand failed with args '%v': %w, error message: %s", args, err, stderr.String())
	}
	if cfg.Verbose && stderr.Len() != 0 {
		cfg.Logger.Info("go subcommand error", zap.String("message", stderr.String()))
	}

	return stdout.Bytes(), nil
}

// GenerateAndCompile will generate the source files based on the given configuration, update go mod, and will compile into a binary
func GenerateAndCompile(cfg Config) error {
	if err := Generate(cfg); err != nil {
		return err
	}

	// run go get to update go.mod and go.sum files
	if err := GetModules(cfg); err != nil {
		return err
	}

	return Compile(cfg)
}

// Generate assembles a new distribution based on the given configuration
func Generate(cfg Config) error {
	if cfg.SkipGenerate {
		cfg.Logger.Info("Skipping generating source codes.")
		return nil
	}
	// create a warning message for non-aligned builder and collector base
	if cfg.Distribution.OtelColVersion != defaultOtelColVersion {
		cfg.Logger.Info("You're building a distribution with non-aligned version of the builder. Compilation may fail due to API changes. Please upgrade your builder or API", zap.String("builder-version", defaultOtelColVersion))
	}
	// if the file does not exist, try to create it
	if _, err := os.Stat(cfg.Distribution.OutputPath); os.IsNotExist(err) {
		if err = os.Mkdir(cfg.Distribution.OutputPath, 0750); err != nil {
			return fmt.Errorf("failed to create output path: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to create output path: %w", err)
	}

	for _, tmpl := range []*template.Template{
		mainTemplate,
		mainOthersTemplate,
		mainWindowsTemplate,
		componentsTemplate,
		goModTemplate,
	} {
		if err := processAndWrite(cfg, tmpl, tmpl.Name(), cfg); err != nil {
			return fmt.Errorf("failed to generate source file %q: %w", tmpl.Name(), err)
		}
	}

	cfg.Logger.Info("Sources created", zap.String("path", cfg.Distribution.OutputPath))
	return nil
}

// Compile generates a binary from the sources based on the configuration
func Compile(cfg Config) error {
	if cfg.SkipCompilation {
		cfg.Logger.Info("Generating source codes only, the distribution will not be compiled.")
		return nil
	}

	cfg.Logger.Info("Compiling")

	var ldflags = "-s -w"

	args := []string{"build", "-trimpath", "-o", cfg.Distribution.Name}
	if cfg.Distribution.DebugCompilation {
		cfg.Logger.Info("Debug compilation is enabled, the debug symbols will be left on the resulting binary")
		ldflags = cfg.LDFlags
		args = append(args, "-gcflags=all=-N -l")
	} else if len(cfg.LDFlags) > 0 {
		ldflags += " " + cfg.LDFlags
	}
	args = append(args, "-ldflags="+ldflags)
	if cfg.Distribution.BuildTags != "" {
		args = append(args, "-tags", cfg.Distribution.BuildTags)
	}
	if _, err := runGoCommand(cfg, args...); err != nil {
		return fmt.Errorf("%w: %s", errCompileFailed, err.Error())
	}
	cfg.Logger.Info("Compiled", zap.String("binary", fmt.Sprintf("%s/%s", cfg.Distribution.OutputPath, cfg.Distribution.Name)))

	return nil
}

// GetModules retrieves the go modules, updating go.mod and go.sum in the process
func GetModules(cfg Config) error {
	if cfg.SkipGetModules {
		cfg.Logger.Info("Generating source codes only, will not update go.mod and retrieve Go modules.")
		return nil
	}

	if _, err := runGoCommand(cfg, "mod", "tidy", "-compat=1.21"); err != nil {
		return fmt.Errorf("failed to update go.mod: %w", err)
	}

	if cfg.SkipStrictVersioning {
		return downloadModules(cfg)
	}

	// Perform strict version checking.  For each component listed and the
	// otelcol core dependency, check that the enclosing go module matches.
	modulePath, dependencyVersions, err := cfg.readGoModFile()
	if err != nil {
		return err
	}

	corePath, coreVersion := cfg.coreModuleAndVersion()
	coreDepVersion, ok := dependencyVersions[corePath]
	if !ok {
		return fmt.Errorf("core collector %w: '%s'. %s", ErrDepNotFound, corePath, skipStrictMsg)
	}
	if semver.MajorMinor(coreDepVersion) != semver.MajorMinor(coreVersion) {
		return fmt.Errorf(
			"%w: core collector version calculated by component dependencies %q does not match configured version %q. %s",
			ErrVersionMismatch, coreDepVersion, coreVersion, skipStrictMsg)
	}

	for _, mod := range cfg.allComponents() {
		module, version, _ := strings.Cut(mod.GoMod, " ")
		if module == modulePath {
			// No need to check the version of components that are part of the
			// module we're building from.
			continue
		}

		moduleDepVersion, ok := dependencyVersions[module]
		if !ok {
			return fmt.Errorf("component %w: '%s'. %s", ErrDepNotFound, module, skipStrictMsg)
		}
		if semver.MajorMinor(moduleDepVersion) != semver.MajorMinor(version) {
			return fmt.Errorf(
				"%w: component %q version calculated by dependencies %q does not match configured version %q. %s",
				ErrVersionMismatch, module, moduleDepVersion, version, skipStrictMsg)
		}
	}

	return downloadModules(cfg)
}

func downloadModules(cfg Config) error {
	cfg.Logger.Info("Getting go modules")
	failReason := "unknown"
	for i := 1; i <= cfg.downloadModules.numRetries; i++ {
		if _, err := runGoCommand(cfg, "mod", "download"); err != nil {
			failReason = err.Error()
			cfg.Logger.Info("Failed modules download", zap.String("retry", fmt.Sprintf("%d/%d", i, cfg.downloadModules.numRetries)))
			time.Sleep(cfg.downloadModules.wait)
			continue
		}
		return nil
	}
	return fmt.Errorf("%w: %s", errDownloadFailed, failReason)
}

func processAndWrite(cfg Config, tmpl *template.Template, outFile string, tmplParams any) error {
	out, err := os.Create(filepath.Clean(filepath.Join(cfg.Distribution.OutputPath, outFile)))
	if err != nil {
		return err
	}

	defer out.Close()
	return tmpl.Execute(out, tmplParams)
}

func (c *Config) coreModuleAndVersion() (string, string) {
	module := "go.opentelemetry.io/collector"
	if c.Distribution.RequireOtelColModule {
		module += "/otelcol"
	}
	return module, "v" + c.Distribution.OtelColVersion
}

func (c *Config) allComponents() []Module {
	// TODO: Use slices.Concat when we drop support for Go 1.21
	return append(c.Exporters,
		append(c.Receivers,
			append(c.Processors,
				append(c.Extensions,
					append(c.Connectors,
						*c.Providers...)...)...)...)...)
}

func (c *Config) readGoModFile() (string, map[string]string, error) {
	var modPath string
	stdout, err := runGoCommand(*c, "mod", "edit", "-print")
	if err != nil {
		return modPath, nil, err
	}
	parsedFile, err := modfile.Parse("go.mod", stdout, nil)
	if err != nil {
		return modPath, nil, err
	}
	if parsedFile != nil && parsedFile.Module != nil {
		modPath = parsedFile.Module.Mod.Path
	}
	dependencies := map[string]string{}
	for _, req := range parsedFile.Require {
		if req == nil {
			continue
		}
		dependencies[req.Mod.Path] = req.Mod.Version
	}
	return modPath, dependencies, nil
}
