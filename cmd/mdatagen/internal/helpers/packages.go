// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package helpers // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/helpers"

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RootPackage determines the root Go module path by finding the highest go.mod above componentDir
// and running "go list -m" in that directory. This is used to resolve local absolute references
// (e.g., "/config/confighttp.client_config") into full Go import paths.
func RootPackage(componentDir string) (string, error) {
	rootModDir, err := rootModuleDir(componentDir)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("go", "list", "-m")
	cmd.Dir = rootModDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to resolve root module path: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func rootModuleDir(componentDir string) (string, error) {
	absDir, err := filepath.Abs(componentDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	var found string
	dir := absDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			found = dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if found == "" {
		return "", fmt.Errorf("no go.mod found in any parent of %s", componentDir)
	}
	return found, nil
}
