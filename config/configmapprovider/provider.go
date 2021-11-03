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

package configmapprovider // import "go.opentelemetry.io/collector/config/configmapprovider"

import (
	"context"

	"go.opentelemetry.io/collector/config"
)

// Provider is an interface that helps providing configuration's parser.
// Implementations may load the parser from a file, a database or any other source.
type Provider interface {
	// Retrieve goes to the configuration source and retrieves the selected data which
	// contains the value to be injected in the configuration and the corresponding watcher that
	// will be used to monitor for updates of the retrieved value.
	// Should never be called concurrently with itself.
	Retrieve(ctx context.Context) (Retrieved, error)

	// Close signals that the configuration for which it was used to retrieve values is no longer in use
	// and the object should close and release any watchers that it may have created.
	// This method must be called when the service ends, either in case of success or error.
	// Should never be called concurrently with itself.
	Close(ctx context.Context) error

	// Note that Retrieve() and Close() should never be called concurrently. Close()
	// should be called only after Retrieve() returns.
}

// Retrieved holds the result of a call to the Retrieve method of a Session object.
type Retrieved interface {
	// Get returns the Map.
	// Should never be called concurrently with itself.
	Get() *config.Map
}

// WatchableRetrieved is an extension for Retrieved that if implemented,
// the Retrieved value supports monitoring for updates.
//
// The typical usage is the following:
//
//	goroutine 1                                        | go routine 2
//	                                                   |
//	r := mapProvider.Retrieve()                        |
//	r.Get()                                            |
//	r.WatchForUpdate() blocks until there are updates  |
//	...                                                |
//	r.Get()                                            |
//	r.WatchForUpdate()                                 | Close() can be called anytime after Retrieve() returns
//  WatchForUpdate() returns when Close() is called.   |
type WatchableRetrieved interface {
	Retrieved

	// WatchForUpdate waits for updates on any of the values retrieved from config sources.
	// It blocks until the configuration updates.
	//
	// If the update is successful then the returned error will be nil.
	// If Close() was called while WatchForUpdate() is in progress then the returned
	// error will be configsource.ErrSessionClosed.
	// If anything else fails during watching some other error may be returned.
	//
	// WatchForUpdate can be used to watch configuration changes continuously.
	//
	// Should never be called concurrently with itself.
	// Should never be called concurrently with Get().
	WatchForUpdate() error

	// Close signals that the configuration for which it was used to retrieve values is
	// no longer in use and the object should close and release any watchers that it
	// may have created.
	//
	// This method must be called when the service ends, either in case of success or error.
	//
	// The method may be called while WatchForUpdate() class is in progress and is blocked.
	// In that case WatchForUpdate() method must abort as soon as possible and
	// return ErrSessionClosed.
	//
	// Should never be called concurrently with itself.
	// May be called before, after or concurrently with WatchForUpdate().
	// May be called before, after or concurrently with Get().
	//
	// Calling Close() on an object that is already closed because WatchForUpdate()
	// has already returned should not block and should return nil.
	Close(ctx context.Context) error
}
