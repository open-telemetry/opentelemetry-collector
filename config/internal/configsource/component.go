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

	// Watcher method returns a Watcher object that will be used to monitor for updates on the
	// items used during the configuration session. The code using the Session object must
	// guarantee that there won't be calls to Apply after the Watcher method is called.
	// The Watcher method must be called only once for a Session object. This method must not
	// be called in case any call to Apply return errors during the configuration session.
	Watcher() (Watcher, error)

	// Close signals that the object won't be used anymore to inject data into a configuration.
	// This method must be called when the configuration session ends, either in case of success
	// or error. Each Session object should use this call according to their needs: release resources,
	// close communication channels, etc.
	Close(ctx context.Context) error
}

// Watcher provides an way for configuration sources to notify when
// updates to the configuration in use were detected. A Watcher must be
// created at the end of a successful configuration session from a Session object
// via the Watcher method.
type Watcher interface {
	// WatchForUpdate must not return until one of the following happens:
	//
	// 1. An update is detected for at least one of the items that were requested in
	//    the original configuration session.
	//
	// 2. The stopWatchingCh channel is closed.
	//
	// 3. An error happens while watching for updates. The method should not return
	//    on first instances of transient errors, optionally there should be
	//    configurable thresholds to control for long such errors can be ignored.
	//
	// The method must return with a nil error when an update has happened to
	// any of the items used during the configuration session from which the watcher
	// object was created. Optionally the object can also return a string with some
	// information about the item that was updated.
	WatchForUpdate(stopWatchingCh <-chan struct{}) (string, error)
}
