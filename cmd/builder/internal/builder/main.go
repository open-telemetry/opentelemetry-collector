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
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

	"go.uber.org/zap"
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
		componentsTestTemplate,
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
	// #nosec G204 -- cfg.Distribution.Go is trusted to be a safe path and the caller is  assumed to have carried out necessary input validation
	cmd := exec.Command(cfg.Distribution.Go, args...)
	cmd.Dir = cfg.Distribution.OutputPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to compile the OpenTelemetry Collector distribution: %w. Output:\n%s", err, out)
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

	// #nosec G204 -- cfg.Distribution.Go is trusted to be a safe path
	cmd := exec.Command(cfg.Distribution.Go, "mod", "tidy", "-compat=1.19")
	cmd.Dir = cfg.Distribution.OutputPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update go.mod: %w. Output:\n%s", err, out)
	}

	cfg.Logger.Info("Getting go modules")
	// basic retry if error from go mod command (in case of transient network error). This could be improved
	// retry 3 times with 5 second spacing interval
	retries := 3
	failReason := "unknown"
	for i := 1; i <= retries; i++ {
		// #nosec G204
		cmd := exec.Command(cfg.Distribution.Go, "mod", "download")
		cmd.Dir = cfg.Distribution.OutputPath
		if out, err := cmd.CombinedOutput(); err != nil {
			failReason = fmt.Sprintf("%s. Output:\n%s", err, out)
			cfg.Logger.Info("Failed modules download", zap.String("retry", fmt.Sprintf("%d/%d", i, retries)))
			time.Sleep(5 * time.Second)
			continue
		}
		return nil
	}
	return fmt.Errorf("failed to download go modules: %s", failReason)
}

func processAndWrite(cfg Config, tmpl *template.Template, outFile string, tmplParams any) error {
	out, err := os.Create(filepath.Clean(filepath.Join(cfg.Distribution.OutputPath, outFile)))
	if err != nil {
		return err
	}

	return tmpl.Execute(out, tmplParams)
}
