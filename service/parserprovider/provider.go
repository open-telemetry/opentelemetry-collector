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

import "go.opentelemetry.io/collector/config"

// ParserProvider is an interface that helps providing configuration's parser.
// Implementations may load the parser from a file, a database or any other source.
type ParserProvider interface {
	// Get returns the config.Parser if succeed or error otherwise.
	Get() (*config.Parser, error)
}
