// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package create // import "go.opentelemetry.io/collector/cmd/builder/internal/create"

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
)

const defaultDescription = "A custom OpenTelemetry Collector exporter component"

//go:embed templates/*.tmpl
var templatesFS embed.FS

type metadata struct {
	Name          string
	Description   string
	StableVersion string
	BetaVersion   string
	HasTraces     bool
	HasMetrics    bool
	HasLogs       bool
	HasProfiles   bool
}

func newExporterCommand() *cobra.Command {
	var signals []string
	var outputPath string

	cmd := &cobra.Command{
		Use:   "exporter",
		Short: "[EXPERIMENTAL] Initializes a new custom exporter repository in the provided folder",
		Long:  `ocb create exporter initializes a new repository in the provided folder with a manifest to start building a custom Collector exporter. This command is experimental and very likely to change.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dirName := args[0]
			if dirName == "" {
				return fmt.Errorf("exporter directory name cannot be empty")
			}

			return runCreateExporter(dirName, signals, outputPath)
		},
	}

	cmd.Flags().StringSliceVar(&signals, "signals", []string{"metrics", "traces", "logs", "profiles"}, "Signals to support: metrics, traces, logs, profiles")
	cmd.Flags().StringVar(&outputPath, "output-path", ".", "Path where the exporter directory will be created")

	return cmd
}

func runCreateExporter(dirName string, signals []string, outputPath string) error {
	if dirName == "" {
		return errors.New("directory name cannot be empty")
	}

	if !strings.HasSuffix(dirName, "exporter") {
		dirName += "exporter"
	}

	fullPath := filepath.Join(outputPath, dirName)

	fullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	err = os.MkdirAll(fullPath, 0o750)
	if err != nil {
		return fmt.Errorf("failed to create folder %s: %w", fullPath, err)
	}

	meta := metadata{
		Name:          dirName,
		Description:   defaultDescription,
		StableVersion: builder.DefaultStableOtelColVersion,
		BetaVersion:   builder.DefaultBetaOtelColVersion,
		HasTraces:     containsSignal(signals, "traces"),
		HasMetrics:    containsSignal(signals, "metrics"),
		HasLogs:       containsSignal(signals, "logs"),
		HasProfiles:   containsSignal(signals, "profiles"),
	}

	err = writeTemplate(fullPath, ".gitignore", meta)
	if err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	err = writeTemplate(fullPath, "go.mod", meta)
	if err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	err = writeTemplate(fullPath, "README.md", meta)
	if err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	err = writeTemplate(fullPath, "Makefile", meta)
	if err != nil {
		return fmt.Errorf("failed to write Makefile: %w", err)
	}

	err = writeTemplate(fullPath, "metadata.yaml", meta)
	if err != nil {
		return fmt.Errorf("failed to write metadata.yaml: %w", err)
	}

	err = writeTemplate(fullPath, "config.go", meta)
	if err != nil {
		return fmt.Errorf("failed to write config.go: %w", err)
	}

	err = writeTemplate(fullPath, "factory.go", meta)
	if err != nil {
		return fmt.Errorf("failed to write factory.go: %w", err)
	}

	err = writeTemplate(fullPath, "exporter.go", meta)
	if err != nil {
		return fmt.Errorf("failed to write exporter.go: %w", err)
	}

	err = runTidy(fullPath)
	if err != nil {
		return fmt.Errorf("failed to run go mod tidy: %w", err)
	}

	return nil
}

func writeTemplate(path, fn string, m metadata) error {
	outputFile := filepath.Join(path, fn)

	content, err := executeTemplate(fn+".tmpl", m)
	if err != nil {
		return err
	}
	return os.WriteFile(outputFile, content, 0o600)
}

func executeTemplate(tmplFile string, m metadata) ([]byte, error) {
	tmplPath := path.Join("templates", tmplFile)
	tmpl := template.Must(template.New(tmplFile).ParseFS(templatesFS, tmplPath))
	buf := bytes.Buffer{}

	if err := tmpl.Execute(&buf, m); err != nil {
		return []byte{}, fmt.Errorf("failed executing template: %w", err)
	}
	return buf.Bytes(), nil
}

func runTidy(path string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w (%s)", err, string(output))
	}

	return nil
}

func containsSignal(signals []string, signal string) bool {
	return slices.Contains(signals, signal)
}
