// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/builder/internal"

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"

	"go.opentelemetry.io/collector/cmd/builder/internal/builder"
)

const defaultDescription = "Custom OpenTelemetry Collector"

//go:embed init/templates/*.tmpl
var templatesFS embed.FS

type metadata struct {
	Name          string
	Description   string
	StableVersion string
	BetaVersion   string
}

func initCommand() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "[EXPERIMENTAL] Initializes a new custom collector repository in the provided folder",
		Long:  `ocb init initializes a new repository in the provided folder with a manifest to start building a custom Collector. This command is experimental and very likely to change.`,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(outputPath)
		},
	}

	cmd.Flags().StringVar(&outputPath, "path", ".", "Output path where the collector repository will be initialized")

	return cmd
}

func run(path string) error {
	if path == "" {
		return errors.New("argument must be a folder")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %v: %w", path, err)
	}
	err = os.MkdirAll(path, 0o750)
	if err != nil {
		return fmt.Errorf("failed creating folder %v: %w", path, err)
	}

	meta := metadata{
		Name:          filepath.Base(path),
		Description:   defaultDescription,
		StableVersion: builder.DefaultStableOtelColVersion,
		BetaVersion:   builder.DefaultBetaOtelColVersion,
	}

	err = writeTemplate(path, "manifest.yaml", meta)
	if err != nil {
		return fmt.Errorf("failed writing manifest: %w", err)
	}

	err = writeTemplate(path, ".gitignore", meta)
	if err != nil {
		return fmt.Errorf("failed writing gitignore: %w", err)
	}

	err = writeTemplate(path, "README.md", meta)
	if err != nil {
		return fmt.Errorf("failed writing README: %w", err)
	}

	err = writeTemplate(path, "go.mod", meta)
	if err != nil {
		return fmt.Errorf("failed writing go.mod: %w", err)
	}

	err = writeTemplate(path, "Makefile", meta)
	if err != nil {
		return fmt.Errorf("failed writing Makefile: %w", err)
	}

	err = writeTemplate(path, "config.yaml", meta)
	if err != nil {
		return fmt.Errorf("failed writing config.yaml: %w", err)
	}

	err = os.MkdirAll(filepath.Join(path, "build"), 0o750)
	if err != nil {
		return fmt.Errorf("failed creating build folder: %w", err)
	}

	err = runTidy(path)
	if err != nil {
		return fmt.Errorf("failed running go mod tidy: %w", err)
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
	tmplPath := path.Join("init/templates", tmplFile)
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
