// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cfgen // import "go.opentelemetry.io/collector/cmd/mdatagen/internal/cfgen"

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
)

type Ref struct {
	Namespace string
	Path      string
	Type      string
}

func (r *Ref) PkgPath() string {
	return filepath.Join(r.Namespace, r.Path)
}

var namespaceToURL = map[string]string{
	"go.opentelemetry.io/collector":                             "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector",
	"github.com/open-telemetry/opentelemetry-collector-contrib": "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector-contrib",
}

var supportedNamespaces = slices.Collect(maps.Keys(namespaceToURL))

func isInNamespace(packagePath string) bool {
	_, err := getNamespace(packagePath)
	return err == nil
}

// getNamespace checks if the given package path belongs to any of the supported namespaces
// and returns the matching namespace.
func getNamespace(packagePath string) (string, error) {
	for _, ns := range supportedNamespaces {
		if strings.HasPrefix(packagePath, ns) {
			return ns, nil
		}
	}
	return "", fmt.Errorf("namespace not supported: %s", packagePath)
}

// getRef parses a reference path and extracts the namespace, version, path, and type information.
func getRef(refPath string) (*Ref, error) {
	namespace, err := getNamespace(refPath)
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimPrefix(refPath, namespace+"/")
	parts := strings.SplitN(trimmed, ".", 2)
	// We expect at least a package path and type name (e.g., "scraper/scraperhelper.controller_config")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid reference path: %s", refPath)
	}
	path := parts[0]
	typ := parts[1]

	return &Ref{
		Namespace: namespace,
		Path:      path,
		Type:      typ,
	}, nil
}

func getRefURL(ref Ref, version string) string {
	// Construct the URL for fetching the reference based on the namespace, version, path
	baseURL := namespaceToURL[ref.Namespace]
	return fmt.Sprintf("%s/%s/%s/config.yaml",
		baseURL,
		version,
		ref.Path)
}
