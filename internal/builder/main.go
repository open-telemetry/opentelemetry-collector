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

	"github.com/open-telemetry/opentelemetry-collector-builder/internal/scaffold"
)

var (
	// ErrFailedToGenerateSourceFromTemplate indicates a failure in writing the final contents to the destination file
	ErrFailedToGenerateSourceFromTemplate = errors.New("failed to generate source from template")

	// ErrGoNotFound is returned when a Go binary hasn't been found
	ErrGoNotFound = errors.New("Go binary not found")
)

// GenerateAndCompile will generate the source files based on the given configuration and will compile it into a binary
func GenerateAndCompile(cfg Config) error {
	if err := Generate(cfg); err != nil {
		return err
	}

	return Compile(cfg)
}

// Generate assembles a new distribution based on the given configuration
func Generate(cfg Config) error {
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
	goBinary := cfg.Distribution.Go
	// first, we test to check if we have Go at all
	if _, err := exec.Command(goBinary, "env").CombinedOutput(); err != nil {
		path, err := exec.LookPath("go")
		if err != nil {
			return ErrGoNotFound
		}
		goBinary = path
		cfg.Logger.Info("Using go from PATH", "Go executable", path)
	}

	cfg.Logger.Info("Compiling")
	cmd := exec.Command(goBinary, "build", "-ldflags=-s -w", "-trimpath", "-o", cfg.Distribution.ExeName)
	cmd.Dir = cfg.Distribution.OutputPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to compile the OpenTelemetry Collector distribution: %w. Output: %q", err, out)
	}
	cfg.Logger.Info("Compiled", "binary", fmt.Sprintf("%s/%s", cfg.Distribution.OutputPath, cfg.Distribution.ExeName))

	return nil
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
