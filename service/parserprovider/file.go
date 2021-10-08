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

package parserprovider // import "go.opentelemetry.io/collector/service/parserprovider"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/config"
)

type fileMapProvider struct {
	fileName string
}

// NewFileMapProvider returns a new config.MapProvider that reads the configuration from the given file.
func NewFileMapProvider(fileName string) config.MapProvider {
	return &fileMapProvider{
		fileName: fileName,
	}
}

func (fmp *fileMapProvider) Get(context.Context) (*config.Map, error) {
	if fmp.fileName == "" {
		return nil, errors.New("config file not specified")
	}

	cp, err := config.NewMapFromFile(fmp.fileName)
	if err != nil {
		return nil, fmt.Errorf("error loading config file %q: %w", fmp.fileName, err)
	}

	return cp, nil
}

func (*fileMapProvider) Close(context.Context) error {
	return nil
}
