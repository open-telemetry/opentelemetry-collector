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

// Provider is an interface that helps to retrieve a config map and watch for any
// changes to the config map. Implementations may load the config from a file,
// a database or any other source.
type Provider interface {
	// Retrieve goes to the configuration source and retrieves the selected data which
	// contains the value to be injected in the configuration and the corresponding watcher that
	// will be used to monitor for updates of the retrieved value.
	//
	// onChange callback is called when the config changes. onChange may be called from
	// a different go routine. After onChange is called Retrieved.Get should be called
	// to get the new config. See description of Retrieved for more details.
	// onChange may be nil, which indicates that the caller is not interested in
	// knowing about the changes.
	//
	// If ctx is cancelled should return immediately with an error.
	// Should never be called concurrently with itself or with Shutdown.
	Retrieve(ctx context.Context, onChange func(*ChangeEvent)) (Retrieved, error)

	// Shutdown signals that the configuration for which this Provider was used to
	// retrieve values is no longer in use and the Provider should close and release
	// any resources that it may have created.
	//
	// This method must be called when the Collector service ends, either in case of
	// success or error. Retrieve cannot be called after Shutdown.
	//
	// Should never be called concurrently with itself or with Retrieve.
	// If ctx is cancelled should return immediately with an error.
	Shutdown(ctx context.Context) error
}

// Retrieved holds the result of a call to the Retrieve method of a Provider object.
//
// The typical usage is the following:
//
//		r := mapProvider.Retrieve()
//		r.Get()
//		// wait for onChange() to be called.
//		r.Close()
//		r = mapProvider.Retrieve()
//		r.Get()
//		// wait for onChange() to be called.
//		r.Close()
//		// repeat Retrieve/Get/wait/Close cycle until it is time to shut down the Collector process.
//		// ...
//		mapProvider.Shutdown()
type Retrieved interface {
	// Get returns the config Map.
	// If Close is called before Get or concurrently with Get then Get
	// should return immediately with ErrSessionClosed error.
	// Should never be called concurrently with itself.
	// If ctx is cancelled should return immediately with an error.
	Get(ctx context.Context) (*config.Map, error)

	// Close signals that the configuration for which it was used to retrieve values is
	// no longer in use and the object should close and release any watchers that it
	// may have created.
	//
	// This method must be called when the service ends, either in case of success or error.
	//
	// Should never be called concurrently with itself.
	// May be called before, after or concurrently with Get.
	// If ctx is cancelled should return immediately with an error.
	//
	// Calling Close on an already closed object should have no effect and should return nil.
	Close(ctx context.Context) error
}

// ChangeEvent describes the particular change event that happened with the config.
// TODO: see if this can be eliminated.
type ChangeEvent struct {
	// Error is nil if the config is changed and needs to be re-fetched using Get.
	// It is set to configsource.ErrSessionClosed if Close was called.
	// Any other error indicates that there was a problem with retrieving the config.
	Error error
}
