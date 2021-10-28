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
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-builder/internal/scaffold"
)

var (
	// ErrGoNotFound is returned when a Go binary hasn't been found
	ErrGoNotFound = errors.New("go binary not found")
)

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
	// create a warning message for non-aligned builder and collector base
	if cfg.Distribution.OtelColVersion != defaultOtelColVersion {
		cfg.Logger.Info("You're building a distribution with non-aligned version of the builder. Compilation may fail due to API changes. Please upgrade your builder or API", "builder-version", defaultOtelColVersion)
	}
	// if the file does not exist, try to create it
	if _, err := os.Stat(cfg.Distribution.OutputPath); os.IsNotExist(err) {
		if err := os.Mkdir(cfg.Distribution.OutputPath, 0755); err != nil {
			return fmt.Errorf("failed to create output path: %w", err)
		}
	} else if err != nil {
		// something else happened
		return fmt.Errorf("failed to create output path: %w", err)
	}

	for _, file := range []struct {
		outFile string
		tmpl    string
	}{
		{
			"main.go",
			scaffold.Main,
		},
		{
			"main_others.go",
			scaffold.MainOthers,
		},
		{
			"main_windows.go",
			scaffold.MainWindows,
		},
		// components.go
		{
			"components.go",
			scaffold.Components,
		},
		{
			"go.mod",
			scaffold.Gomod,
		},
	} {
		if err := processAndWrite(cfg, file.tmpl, file.outFile, cfg); err != nil {
			return fmt.Errorf("failed to generate source file with destination %q, source: %q: %w", file.outFile, file.tmpl, err)
		}
	}

	cfg.Logger.Info("Sources created", "path", cfg.Distribution.OutputPath)
	return nil
}

// Compile generates a binary from the sources based on the configuration
func Compile(cfg Config) error {
	if cfg.SkipCompilation {
		cfg.Logger.Info("Generating source codes only, the distribution will not be compiled.")
		return nil
	}

	cfg.Logger.Info("Compiling")
	// #nosec G204
	cmd := exec.Command(cfg.Distribution.Go, "build", "-ldflags=-s -w", "-trimpath", "-o", cfg.Distribution.ExeName)
	cmd.Dir = cfg.Distribution.OutputPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to compile the OpenTelemetry Collector distribution: %w. Output: %q", err, out)
	}
	cfg.Logger.Info("Compiled", "binary", fmt.Sprintf("%s/%s", cfg.Distribution.OutputPath, cfg.Distribution.ExeName))

	return nil
}

// GetModules retrieves the go modules, updating go.mod and go.sum in the process
func GetModules(cfg Config) error {
	// #nosec G204
	cmd := exec.Command(cfg.Distribution.Go, "mod", "tidy", "-compat=1.17")
	cmd.Dir = cfg.Distribution.OutputPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update go.mod: %w. Output: %q", err, out)
	}

	cfg.Logger.Info("Getting go modules")
	// basic retry if error from go mod command (in case of transient network error). This could be improved
	// retry 3 times with 5 second spacing interval
	retries := 3
	failReason := "unknown"
	for i := 1; i <= retries; i++ {
		// #nosec G204
		cmd := exec.Command(cfg.Distribution.Go, "mod", "download", "all")
		cmd.Dir = cfg.Distribution.OutputPath
		if out, err := cmd.CombinedOutput(); err != nil {
			failReason = fmt.Sprintf("%s. Output: %q", err, out)
			cfg.Logger.Info("Failed modules download", "retry", fmt.Sprintf("%d/%d", i, retries))
			time.Sleep(5 * time.Second)
			continue
		}
		return nil
	}
	return fmt.Errorf("failed to download go modules: %s", failReason)
}

func processAndWrite(cfg Config, tmpl string, outFile string, tmplParams interface{}) error {
	t, err := template.New("template").Parse(tmpl)
	if err != nil {
		return err
	}

	out, err := os.Create(filepath.Join(cfg.Distribution.OutputPath, outFile))
	if err != nil {
		return err
	}

	return t.Execute(out, tmplParams)
}
