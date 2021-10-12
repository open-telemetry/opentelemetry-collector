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

	"go.opentelemetry.io/collector/config"
)

// MapProvider is an interface that helps providing configuration's parser.
// Implementations may load the parser from a file, a database or any other source.
type MapProvider interface {
	// Get returns the config.Map if succeed or error otherwise.
	Get(ctx context.Context) (*config.Map, error)

	// Close signals that the configuration for which it was used to retrieve values is no longer in use
	// and the object should close and release any watchers that it may have created.
	// This method must be called when the service ends, either in case of success or error.
	Close(ctx context.Context) error
}

// Watchable is an extension for MapProvider that is implemented if the given provider
// supports monitoring of configuration updates.
type Watchable interface {
	// WatchForUpdate waits for updates on any of the values retrieved from config sources.
	// It blocks until configuration updates are received and can
	// return an error if anything fails. WatchForUpdate is used once during the
	// first evaluation of the configuration and is not used to watch configuration
	// changes continuously.
	WatchForUpdate() error
}
