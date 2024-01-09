// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder // import "go.opentelemetry.io/collector/cmd/builder/internal/builder"

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"go.opentelemetry.io/collector/cmd/builder/internal/builder/modfile"
	"go.uber.org/zap"
	"go.uber.org/zap/zapio"
	"golang.org/x/mod/semver"
)

var (
	// ErrGoNotFound is returned when a Go binary hasn't been found
	ErrGoNotFound = errors.New("go binary not found")
)

func runGoCommandStdout(cfg Config, args ...string) ([]byte, error) {
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
		return nil, fmt.Errorf("go subcommand failed with args '%v': %w.", args, err)
	}
	if stderr.Len() != 0 {
		cfg.Logger.Info("go subcommand failed", zap.Strings("args", args), zap.String("stderr", stderr.String()))
	}

	return stdout.Bytes(), nil
}

func runGoCommand(cfg Config, args ...string) error {
	cfg.Logger.Info("Running go subcommand.", zap.Any("arguments", args))
	// #nosec G204 -- cfg.Distribution.Go is trusted to be a safe path and the caller is assumed to have carried out necessary input validation
	cmd := exec.Command(cfg.Distribution.Go, args...)
	cmd.Dir = cfg.Distribution.OutputPath

	if cfg.Verbose {
		writer := &zapio.Writer{Log: cfg.Logger}
		defer func() { _ = writer.Close() }()
		cmd.Stdout = writer
		cmd.Stderr = writer
		return cmd.Run()
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go subcommand failed with args '%v': %w. Output:\n%s", args, err, out)
	}

	return nil
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

	allTemplates := []*template.Template{
		mainTemplate,
		mainOthersTemplate,
		mainWindowsTemplate,
		componentsTemplate,
		componentsTestTemplate,
	}

	if !cfg.SkipNewGoModule {
		allTemplates = append(allTemplates, goModTemplate)
	}

	for _, tmpl := range allTemplates {
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
	if err := runGoCommand(cfg, args...); err != nil {
		return fmt.Errorf("failed to compile the OpenTelemetry Collector distribution: %w", err)
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

	// ambiguous import: found package cloud.google.com/go/compute/metadata in multiple modules
	if err := runGoCommand(cfg, "get", "cloud.google.com/go"); err != nil {
		return fmt.Errorf("failed to go get: %w", err)
	}

	if err := cfg.updateModules(); err != nil {
		return err
	}

	if err := runGoCommand(cfg, "mod", "tidy", "-compat=1.20"); err != nil {
		return fmt.Errorf("failed to update go.mod: %w", err)
	}

	cfg.Logger.Info("Getting go modules")
	// basic retry if error from go mod command (in case of transient network error). This could be improved
	// retry 3 times with 5 second spacing interval
	retries := 3
	failReason := "unknown"
	for i := 1; i <= retries; i++ {
		if err := runGoCommand(cfg, "mod", "download"); err != nil {
			failReason = err.Error()
			cfg.Logger.Info("Failed modules download", zap.String("retry", fmt.Sprintf("%d/%d", i, retries)))
			time.Sleep(5 * time.Second)
			continue
		}
		return nil
	}
	return fmt.Errorf("failed to download go modules: %s", failReason)
}

func (c *Config) updateModules() error {
	if !c.SkipNewGoModule {
		return nil
	}

	// Build the main service dependency
	corespec := "go.opentelemetry.io/collector"
	if c.Distribution.RequireOtelColModule {
		corespec += "/otelcol"
	}
	corespec += " v"
	corespec += c.Distribution.OtelColVersion

	if err := c.updateGoModule(corespec); err != nil {
		return err
	}

	for _, comp := range append(c.Exporters,
		append(c.Receivers,
			append(c.Processors,
				append(c.Extensions,
					c.Connectors...)...)...)...) {
		if err := c.updateGoModule(comp.GoMod); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) updateGoModule(modspec string) error {
	// Re-parse the go.mod file on each iteration, since it can change
	// each time.
	var mf modfile.GoMod

	stdout, err := runGoCommandStdout(*c, "mod", "edit", "-json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(stdout, &mf); err != nil {
		return err
	}
	mvm := map[string]string{}
	for _, req := range mf.Require {
		mvm[req.Path] = req.Version
	}

	mod, ver, found := strings.Cut(modspec, " ")
	if !found {
		return fmt.Errorf("go.mod syntax is 'modulename version', no separator found: %q", modspec)
	}
	if mod == mf.Module.Path {
		// this component is part of the same module, nothing to update.
		return nil
	}

	// check for exact match
	hasVer, ok := mvm[mod]
	if ok && hasVer == ver {
		c.Logger.Info("Component version match", zap.String("module", mod), zap.String("version", ver))
		return nil
	}

	scomp := semver.Compare(hasVer, ver)
	if scomp > 0 {
		// version in enclosing module is newer, do not change
		c.Logger.Info("Not upgrading component, enclosing module is newer.", zap.String("module", mod), zap.String("existing", hasVer), zap.String("config_version", ver))
		return nil
	}

	// upgrading or changing version
	updatespec := mod + "@" + ver

	if err := runGoCommand(*c, "get", updatespec); err != nil {
		return err
	}
	return nil
}

func processAndWrite(cfg Config, tmpl *template.Template, outFile string, tmplParams any) error {
	out, err := os.Create(filepath.Clean(filepath.Join(cfg.Distribution.OutputPath, outFile)))
	if err != nil {
		return err
	}

	return tmpl.Execute(out, tmplParams)
}
