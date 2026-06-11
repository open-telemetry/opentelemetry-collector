// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder // import "go.opentelemetry.io/collector/cmd/builder/internal/builder"

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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

const otelcolPath = "go.opentelemetry.io/collector/otelcol"

func runGoCommand(cfg *Config, args ...string) ([]byte, error) {
	//nolint:gosec // #nosec G204 -- cfg.Distribution.Go is trusted to be a safe path and the caller is assumed to have carried out necessary input validation
	cmd := exec.Command(cfg.Distribution.Go, args...)
	cmd.Dir = cfg.Distribution.OutputPath

	cmd.Env = os.Environ()
	if cfg.Distribution.CGoEnabled {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=1")
	} else {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=0")
	}

	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	if err := cmd.Start(); err != nil {
		stdoutReader.Close()
		stderrReader.Close()
		stdoutWriter.Close()
		stderrWriter.Close()
		return nil, fmt.Errorf("failed to start go subcommand with args '%v': %w", args, err)
	}
	if cfg.Verbose {
		cfg.Logger.Info("Started go subcommand.", zap.Any("arguments", args), zap.Int("pid", cmd.Process.Pid))
	}

	logger := func(reader *io.PipeReader, outputType string, output *bytes.Buffer) {
		defer reader.Close()
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line)
			output.WriteByte('\n')
			if cfg.Verbose {
				cfg.Logger.Info(
					"Go subcommand output",
					zap.Int("pid", cmd.Process.Pid),
					zap.String("type", outputType),
					zap.String("line", line),
				)
			}
		}
		if err := scanner.Err(); err != nil {
			cfg.Logger.Error(
				"Failed to read go subcommand output",
				zap.Int("pid", cmd.Process.Pid),
				zap.String("type", outputType),
				zap.Error(err),
			)
		}
	}

	var wg sync.WaitGroup
	var stderrOutput bytes.Buffer
	var stdoutOutput bytes.Buffer
	wg.Go(func() {
		logger(stderrReader, "stderr", &stderrOutput)
	})
	wg.Go(func() {
		logger(stdoutReader, "stdout", &stdoutOutput)
	})

	err := cmd.Wait()
	stderrWriter.Close()
	stdoutWriter.Close()
	wg.Wait()
	if err != nil {
		return nil, fmt.Errorf("go subcommand failed with args '%v': %w, error message: %s", args, err, stderrOutput.String())
	}
	return stdoutOutput.Bytes(), nil
}

// GenerateAndCompile will generate the source files based on the given configuration, update go mod, and will compile into a binary
func GenerateAndCompile(cfg *Config) error {
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
func Generate(cfg *Config) error {
	if cfg.SkipGenerate {
		cfg.Logger.Info("Skipping generating source codes.")
		return nil
	}

	// if the file does not exist, try to create it
	if _, err := os.Stat(cfg.Distribution.OutputPath); os.IsNotExist(err) {
		if err = os.Mkdir(cfg.Distribution.OutputPath, 0o750); err != nil {
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
		if tmpl == goModTemplate && cfg.SkipGetModules {
			continue
		}
		if err := processAndWrite(cfg, tmpl, tmpl.Name(), cfg); err != nil {
			return fmt.Errorf("failed to generate source file %q: %w", tmpl.Name(), err)
		}
	}

	cfg.Logger.Info("Sources created", zap.String("path", cfg.Distribution.OutputPath))
	return nil
}

// Compile generates a binary from the sources based on the configuration
func Compile(cfg *Config) error {
	if cfg.SkipCompilation {
		cfg.Logger.Info("Generating source codes only, the distribution will not be compiled.")
		return nil
	}

	cfg.Logger.Info("Compiling")
	ldflags := "-s -w" // we strip the symbols by default for smaller binaries
	gcflags := ""
	binaryName := outputBinaryName(cfg.Distribution.Name)

	args := []string{"build", "-trimpath", "-o", binaryName}
	if cfg.Distribution.DebugCompilation {
		cfg.Logger.Info("Debug compilation is enabled, the debug symbols will be left on the resulting binary")
		ldflags = cfg.LDFlags
		gcflags = "all=-N -l"
	} else {
		if cfg.LDSet {
			cfg.Logger.Info("Using custom ldflags", zap.String("ldflags", cfg.LDFlags))
			ldflags = cfg.LDFlags
		}
		if cfg.GCSet {
			cfg.Logger.Info("Using custom gcflags", zap.String("gcflags", cfg.GCFlags))
			gcflags = cfg.GCFlags
		}
	}
	if cfg.Distribution.CGoEnabled {
		cfg.Logger.Info("Building with cgo enabled")
	}

	args = append(args, "-ldflags="+ldflags, "-gcflags="+gcflags)

	if cfg.Distribution.BuildTags != "" {
		args = append(args, "-tags", cfg.Distribution.BuildTags)
	}
	if _, err := runGoCommand(cfg, args...); err != nil {
		return fmt.Errorf("%w: %s", errCompileFailed, err.Error())
	}
	cfg.Logger.Info("Compiled", zap.String("binary", fmt.Sprintf("%s/%s", cfg.Distribution.OutputPath, binaryName)))

	return nil
}

func outputBinaryName(name string) string {
	goos, ok := os.LookupEnv("GOOS")
	if !ok || goos == "" {
		goos = runtime.GOOS
	}

	if !strings.EqualFold(goos, "windows") {
		return name
	}
	if strings.EqualFold(filepath.Ext(name), ".exe") {
		return name
	}
	return name + ".exe"
}

// GetModules retrieves the go modules, updating go.mod and go.sum in the process
func GetModules(cfg *Config) error {
	if cfg.SkipGetModules {
		cfg.Logger.Info("Generating source codes only, will not update go.mod and retrieve Go modules.")
		return nil
	}

	if _, err := runGoCommand(cfg, "mod", "tidy", "-compat=1.25"); err != nil {
		return fmt.Errorf("failed to update go.mod: %w", err)
	}

	if cfg.SkipStrictVersioning {
		return downloadModules(cfg)
	}

	// Perform strict version checking.  For each component listed and the
	// otelcol core dependency, check that the enclosing go module matches.
	modulePath, dependencyVersions, err := readGoModFile(cfg)
	if err != nil {
		return err
	}

	coreDepVersion, ok := dependencyVersions[otelcolPath]
	betaVersion := semver.MajorMinor(DefaultBetaOtelColVersion)
	if !ok {
		return fmt.Errorf("core collector %w: '%s'. %s", ErrDepNotFound, otelcolPath, skipStrictMsg)
	}
	if semver.MajorMinor(coreDepVersion) != betaVersion {
		return fmt.Errorf(
			"%w: core collector version calculated by component dependencies %q does not match configured version %q. %s",
			ErrVersionMismatch, coreDepVersion, betaVersion, skipStrictMsg)
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

func downloadModules(cfg *Config) error {
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

func processAndWrite(cfg *Config, tmpl *template.Template, outFile string, tmplParams any) error {
	out, err := os.Create(filepath.Clean(filepath.Join(cfg.Distribution.OutputPath, outFile)))
	if err != nil {
		return err
	}

	defer out.Close()
	return tmpl.Execute(out, tmplParams)
}

func readGoModFile(cfg *Config) (string, map[string]string, error) {
	var modPath string
	stdout, err := runGoCommand(cfg, "mod", "edit", "-print")
	if err != nil {
		return modPath, nil, err
	}
	parsedFile, err := modfile.Parse("go.mod", stdout, nil)
	if err != nil {
		return modPath, nil, err
	}
	if parsedFile.Module != nil {
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
