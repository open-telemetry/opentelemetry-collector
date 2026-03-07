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
			path := args[0]
			if path == "" {
				return fmt.Errorf("exporter directory name cannot be empty")
			}

			return runCreateExporter(path+"exporter", signals, outputPath)
		},
	}

	cmd.Flags().StringSliceVar(&signals, "signals", []string{"metrics", "traces", "logs"}, "Signals to support: metrics, traces, logs, profiles")
	cmd.Flags().StringVar(&outputPath, "output-path", ".", "Path where the exporter directory will be created")

	return cmd
}

func runCreateExporter(path string, signals []string, outputPath string) error {
	if path == "" {
		return errors.New("argument must be a folder")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	err = os.MkdirAll(path, 0o750)
	if err != nil {
		return fmt.Errorf("failed to create folder %s: %w", path, err)
	}

	err = runTidy(path)
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
