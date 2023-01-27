// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package confmaptest // import "go.opentelemetry.io/collector/confmap/confmaptest"

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"

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
	if err = yaml.Unmarshal(content, &rawConf); err != nil {
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
