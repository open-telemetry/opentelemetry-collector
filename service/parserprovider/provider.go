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

	"go.opentelemetry.io/collector/config/configparser"
)

// ParserProvider is an interface that helps providing configuration's parser.
// Implementations may load the parser from a file, a database or any other source.
type ParserProvider interface {
	// Get returns the config.Parser if succeed or error otherwise.
	Get() (*configparser.Parser, error)
}

// Watchable is an extension for ParserProvider that is implemented if the given provider
// supports monitoring for configuration updates.
type Watchable interface {
	// WatchForUpdate is used to monitor for updates on the retrieved value.
	WatchForUpdate() error
}

// Closeable is an extension interface for ParserProvider that should be added if they need to be closed.
type Closeable interface {
	Close(ctx context.Context) error
}
