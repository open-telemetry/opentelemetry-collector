// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package create // import "go.opentelemetry.io/collector/cmd/builder/internal/create"

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

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

			return runCreateExporter(dirName+"exporter", signals, outputPath)
		},
	}

	cmd.Flags().StringSliceVar(&signals, "signals", []string{"metrics", "traces", "logs"}, "Signals to support: metrics, traces, logs, profiles")
	cmd.Flags().StringVar(&outputPath, "output-path", ".", "Path where the exporter directory will be created")

	return cmd
}

func runCreateExporter(dirName string, signals []string, outputPath string) error {
	if dirName == "" {
		return errors.New("directory name cannot be empty")
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

	err = runTidy(fullPath)
	if err != nil {
		return fmt.Errorf("failed to run go mod tidy: %w", err)
	}

	return nil
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
