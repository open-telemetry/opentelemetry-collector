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
	"errors"
)

// ErrSessionClosed is returned by WatchForUpdate functions when its parent Session
// object is closed.
var ErrSessionClosed = errors.New("parent session was closed")

// ConfigSource is the interface to be implemented by objects used by the collector
// to retrieve external configuration information.
type ConfigSource interface {
	// NewSession must create a Session object that will be used to retrieve data to
	// be injected into a configuration.
	//
	// The Session object should use its creation according to their ConfigSource needs:
	// lock resources, open connections, etc. An implementation, for instance,
	// can use the creation of the Session object to prevent torn configurations,
	// by acquiring a lock (or some other mechanism) that prevents concurrent changes to the
	// configuration during time that data is being retrieved from the source.
	//
	// The code managing the returned Session object must guarantee that the object is not used
	// concurrently and that a single ConfigSource only have one Session open at any time.
	NewSession(ctx context.Context) (Session, error)
}

// Session is the interface used to retrieve configuration data from a ConfigSource. A Session
// object is created from a ConfigSource. The code using Session objects must guarantee that
// methods of a single instance are not called concurrently.
type Session interface {
	// Retrieve goes to the configuration source and retrieves the selected data which
	// contains the value to be injected in the configuration and the corresponding watcher that
	// will be used to monitor for updates of the retrieved value. The retrieved value is selected
	// according to the selector and the params passed in the call to Retrieve.
	//
	// The selector is a string that is required on all invocations, the params are optional. Each
	// implementation handles the generic params according to their requirements.
	Retrieve(ctx context.Context, selector string, params interface{}) (Retrieved, error)

	// RetrieveEnd signals that the Session must not be used to retrieve any new values from the
	// source, ie.: all values from this source were retrieved for the configuration. It should
	// be used to release resources that are only needed to retrieve configuration data.
	RetrieveEnd(ctx context.Context) error

	// Close signals that the configuration for which it was used to retrieve values is no longer in use
	// and the object should close and release any watchers that it may have created.
	// This method must be called when the configuration session ends, either in case of success
	// or error. Each Session object should use this call according to their needs: release resources,
	// close communication channels, etc.
	Close(ctx context.Context) error
}

// Retrieved holds the result of a call to the Retrieve method of a Session object.
type Retrieved struct {
	// Value is the retrieved data that will be injected on the configuration.
	Value interface{}
	// WatchForUpdate must not return until one of the following happens:
	//
	// 1. An update is detected for the monitored value.
	//
	// 2. The parent Session object is closed, in which case the method should return
	//    ErrSessionClosed.
	//
	// 3. An error happens while watching for updates. The method should not return
	//    on first instances of transient errors, optionally there should be
	//    configurable thresholds to control for how long such errors can be ignored.
	//
	// The method must return with a nil error when an update has happened to
	// the value monitored by the Watcher.
	WatchForUpdate func() error
}
