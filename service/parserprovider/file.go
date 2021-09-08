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
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/config/configparser"
)

type fileProvider struct{}

// NewFile returns a new ParserProvider that reads the configuration from a file configured
// via the --config command line flag.
func NewFile() ParserProvider {
	return &fileProvider{}
}

func (fl *fileProvider) Get() (*configparser.ConfigMap, error) {
	fileName := getConfigFlag()
	if fileName == "" {
		return nil, errors.New("config file not specified")
	}

	cp, err := configparser.NewParserFromFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("error loading config file %q: %v", fileName, err)
	}

	return cp, nil
}
