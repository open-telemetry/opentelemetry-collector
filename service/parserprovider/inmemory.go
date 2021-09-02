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
	"io"

	"go.opentelemetry.io/collector/config/configparser"
)

type inMemoryProvider struct {
	buf io.Reader
}

// NewInMemory returns a new ParserProvider that reads the configuration from the provided buffer as YAML.
func NewInMemory(buf io.Reader) ParserProvider {
	return &inMemoryProvider{buf: buf}
}

func (inp *inMemoryProvider) Get() (*configparser.ConfigMap, error) {
	return configparser.NewParserFromBuffer(inp.buf)
}
