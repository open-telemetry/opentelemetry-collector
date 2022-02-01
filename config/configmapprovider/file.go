// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configmapprovider // import "go.opentelemetry.io/collector/config/configmapprovider"

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/config"
)

const fileSchemeName = "file"

type fileMapProvider struct{}

// NewFile returns a new Provider that reads the configuration from a file.
//
// This Provider supports "file" scheme, and can be called with a "location" that follows:
//   file-location = "file:" local-path
//   local-path    = [ drive-letter ] file-path
//   drive-letter  = ALPHA ":"
// The "file-path" can be relative or absolute, and it can be any OS supported format.
//
// Examples:
// `file:path/to/file` - relative path (unix, windows)
// `file:/path/to/file` - absolute path (unix, windows)
// `file:c:/path/to/file` - absolute path including drive-letter (windows)
// `file:c:\path\to\file` - absolute path including drive-letter (windows)
func NewFile() Provider {
	return &fileMapProvider{}
}

func (fmp *fileMapProvider) Retrieve(_ context.Context, location string, _ WatcherFunc) (Retrieved, error) {
	if !strings.HasPrefix(location, fileSchemeName+":") {
		return nil, fmt.Errorf("%v location is not supported by %v provider", location, fileSchemeName)
	}

	// Clean the path before using it.
	content, err := ioutil.ReadFile(filepath.Clean(location[len(fileSchemeName)+1:]))
	if err != nil {
		return nil, fmt.Errorf("unable to read the file %v: %w", location, err)
	}

	var data map[string]interface{}
	if err = yaml.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("unable to parse yaml: %w", err)
	}

	return NewRetrieved(func(ctx context.Context) (*config.Map, error) {
		return config.NewMapFromStringMap(data), nil
	})
}

func (*fileMapProvider) Shutdown(context.Context) error {
	return nil
}
