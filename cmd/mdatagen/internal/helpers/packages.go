// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package helpers // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/helpers"

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RootPackage determines the root Go module path by reading the module directive
// from the go.mod at the repository root. This is used to resolve local absolute
// references (e.g., "/config/confighttp.client_config") into full Go import paths.
func RootPackage(componentDir string) (string, error) {
	repoRoot, err := repoRoot(componentDir)
	if err != nil {
		return "", err
	}

	goModData, err := os.ReadFile(filepath.Clean(filepath.Join(repoRoot, "go.mod")))
	if err != nil {
		return "", fmt.Errorf("failed to read root go.mod: %w", err)
	}

	for line := range strings.SplitSeq(string(goModData), "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(after), nil
		}
	}
	return "", errors.New("module directive not found in root go.mod")
}

func repoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to find repo root: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
