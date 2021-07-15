// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

// Extension is the interface that storage extensions must implement
type Extension interface {
	component.Extension

	// GetClient will create a client for use by the specified component.
	// The component can use the client to manage state
	GetClient(context.Context, component.Kind, config.ComponentID, string) (Client, error)
}

// Client is the interface that storage clients must implement
// All methods should return error only if a problem occurred.
// This mirrors the behavior of a golang map:
//   - Set doesn't error if a key already exists - it just overwrites the value.
//   - Get doesn't error if a key is not found - it just returns nil.
//   - Delete doesn't error if the key doesn't exist - it just no-ops.
// Similarly:
//   - Batch doesn't error if any of the above happens for either retrieved or updated keys
// This also provides a way to differentiate data operations
//   [overwrite | not-found | no-op] from "real" problems
type Client interface {

	// Get will retrieve data from storage that corresponds to the
	// specified key. It should return nil, nil if not found
	Get(context.Context, string) ([]byte, error)

	// Set will store data. The data can be retrieved by the same
	// component after a process restart, using the same key
	Set(context.Context, string, []byte) error

	// Delete will delete data associated with the specified key
	Delete(context.Context, string) error

	// Batch will, respectively - get values for selected keys or upsert key/values. When the value specified
	// is nil, the key is being deleted. It will return an array of results, where each
	// one corresponds to a key at a given position and will be nil, if key is not found
	Batch(context.Context, []string, map[string][]byte) ([][]byte, error)

	// Close will release any resources held by the client
	Close(context.Context) error
}
