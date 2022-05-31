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

package fileprovider // import "go.opentelemetry.io/collector/confmap/provider/fileprovider"

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/confmap"
)

const schemeName = "file"

type provider struct{}

// New returns a new confmap.Provider that reads the configuration from a file.
//
// This Provider supports "file" scheme, and can be called with a "uri" that follows:
//   file-uri		= "file:" local-path
//   local-path		= [ drive-letter ] file-path
//   drive-letter	= ALPHA ":"
// The "file-path" can be relative or absolute, and it can be any OS supported format.
//
// Examples:
// `file:path/to/file` - relative path (unix, windows)
// `file:/path/to/file` - absolute path (unix, windows)
// `file:c:/path/to/file` - absolute path including drive-letter (windows)
// `file:c:\path\to\file` - absolute path including drive-letter (windows)
func New() confmap.Provider {
	return &provider{}
}

func (fmp *provider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return confmap.Retrieved{}, fmt.Errorf("%v uri is not supported by %v provider", uri, schemeName)
	}

	// Clean the path before using it.
	content, err := ioutil.ReadFile(filepath.Clean(uri[len(schemeName)+1:]))
	if err != nil {
		return confmap.Retrieved{}, fmt.Errorf("unable to read the file %v: %w", uri, err)
	}

	var data map[string]interface{}
	if err = yaml.Unmarshal(content, &data); err != nil {
		return confmap.Retrieved{}, fmt.Errorf("unable to parse yaml: %w", err)
	}

	return confmap.NewRetrievedFromMap(confmap.NewFromStringMap(data)), nil
}

func (*provider) Scheme() string {
	return schemeName
}

func (*provider) Shutdown(context.Context) error {
	return nil
}
