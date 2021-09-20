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
	// Retrieve goes to the configuration source and retrieves the selected data which
	// contains the value to be injected in the configuration and the corresponding watcher that
	// will be used to monitor for updates of the retrieved value.
	//
	// The selector is a string that is required on all invocations, the params are optional. Each
	// implementation handles the generic params according to their requirements.
	Retrieve(ctx context.Context) (Retrieved, error)

	// Close signals that the configuration for which it was used to retrieve values is no longer in use
	// and the object should close and release any watchers that it may have created.
	// This method must be called when the service ends, either in case of success or error.
	Close(ctx context.Context) error
}

// Retrieved holds the result of a call to the Retrieve method of a Session object.
type Retrieved interface {
	// Get returns the configparser.ConfigMap if succeed or error otherwise.
	Get() (*configparser.ConfigMap, error)
}

// Watchable is an optional interface that Retrieved can implement if the given source
// supports monitoring for updates.
type Watchable interface {
	// WatchForUpdate waits for updates on any of the values retrieved from config sources.
	// It blocks until configuration updates are received and can
	// return an error if anything fails. WatchForUpdate is used once during the
	// first evaluation of the configuration and is not used to watch configuration
	// changes continuously.
	WatchForUpdate() error
}

// TODO: This probably will make sense to be exported, but needs better name and documentation.
type simpleRetrieved struct {
	confMap *configparser.ConfigMap
}

func (sr *simpleRetrieved) Get() (*configparser.ConfigMap, error) {
	return sr.confMap, nil
}
