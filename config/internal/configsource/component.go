// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configsource

import (
	"context"
)

// ConfigSource is the interface to be implemented by objects used by the collector
// to retrieve external configuration information.
type ConfigSource interface {
	// NewSession must create a Session object that will be used to inject data into
	// a configuration.
	//
	// The Session object should use its creation according to their ConfigSource needs:
	// lock resources, suspend background tasks, etc. An implementation, for instance,
	// can use the creation of the Session object to prevent torn configurations,
	// by acquiring a lock (or some other mechanism) that prevents concurrent changes to the
	// configuration during the middle of a session.
	//
	// The code managing the returned Session object must guarantee that the object is not used
	// concurrently.
	NewSession(ctx context.Context) (Session, error)
}

// Session is the interface used to inject configuration data from a ConfigSource. A Session
// object is created from a ConfigSource. The code using Session objects must guarantee that
// methods of a single instance are not called concurrently.
type Session interface {
	// Apply goes to the configuration source, and according to the specified selector and
	// parameters retrieves a configuration value that is injected into the configuration.
	//
	// The selector is a string that is required on all invocations, the params are optional.
	Apply(ctx context.Context, selector string, params interface{}) (interface{}, error)

	// Close signals that the object won't be used anymore to inject data into a configuration.
	// Each Session object should use this call according to their needs: release resources,
	// close communication channels, etc.
	Close(ctx context.Context) error
}
