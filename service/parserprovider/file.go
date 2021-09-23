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

package parserprovider

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/config"
)

type fileMapProvider struct{}

// NewFileMapProvider returns a new MapProvider that reads the configuration from a file configured
// via the --config command line flag.
func NewFileMapProvider() MapProvider {
	return &fileMapProvider{}
}

func (fl *fileMapProvider) Get(context.Context) (*config.Map, error) {
	fileName := getConfigFlag()
	if fileName == "" {
		return nil, errors.New("config file not specified")
	}

	cp, err := config.NewMapFromFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("error loading config file %q: %w", fileName, err)
	}

	return cp, nil
}

func (fl *fileMapProvider) Close(context.Context) error {
	return nil
}
