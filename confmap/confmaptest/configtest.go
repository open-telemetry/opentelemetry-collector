// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmaptest // import "go.opentelemetry.io/collector/confmap/confmaptest"

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	yaml "go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/confmap"
)

// LoadConf loads a confmap.Conf from file, and does NOT validate the configuration.
func LoadConf(fileName string) (*confmap.Conf, error) {
	// Clean the path before using it.
	content, err := os.ReadFile(filepath.Clean(fileName))
	if err != nil {
		return nil, fmt.Errorf("unable to read the file %v: %w", fileName, err)
	}

	var rawConf map[string]any
	if err := yaml.Unmarshal(content, &rawConf); err != nil {
		return nil, err
	}

	return confmap.NewFromStringMap(rawConf), nil
}

var schemeValidator = regexp.MustCompile("^[A-Za-z][A-Za-z0-9+.-]+$")

// ValidateProviderScheme enforces that given confmap.Provider.Scheme() object is following the restriction defined by the collector:
//   - Checks that the scheme name follows the restrictions defined https://datatracker.ietf.org/doc/html/rfc3986#section-3.1
//   - Checks that the scheme name has at leas two characters per the confmap.Provider.Scheme() comment.
func ValidateProviderScheme(p confmap.Provider) error {
	scheme := p.Scheme()
	if len(scheme) < 2 {
		return fmt.Errorf("scheme must be at least 2 characters long: %q", scheme)
	}

	if !schemeValidator.MatchString(scheme) {
		return fmt.Errorf("scheme names consist of a sequence of characters beginning with a letter and followed by any combination of letters, digits, \"+\", \".\", or \"-\": %q", scheme)
	}

	return nil
}
