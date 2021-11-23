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

// ValueProvider is an interface that helps to retrieve config values. Config values
// can be either a primitive value, such as string or number or a config.Map - a map
// of key/values. One ValueProvider may allow retrieving one or more values from the
// the same source (values are identified by the "selector", see below).
// ValueProvider allows to watch for any changes to the config values.
// Implementations may load the config value from a file, a database or any other source.
type ValueProvider interface {
	Provider

	// RetrieveValue goes to the configuration source and retrieves the selected data
	// which contains the value to be injected in the configuration.
	//
	// onChange callback is called when the config value changes. onChange may be called from
	// a different go routine. After onChange is called Retrieved.Get should be called
	// to get the new config value. See description of RetrievedValue for more details.
	// onChange may be nil, which indicates that the caller is not interested in
	// knowing about the changes.
	//
	// If ctx is cancelled should return immediately with an error.
	// Should never be called concurrently with itself or with Shutdown.
	//
	// selector is an implementation-defined string which identifies which of the
	// values that the provider has to retrieve. For example for a ValueProvider that
	// retrieves values from process environment the selector may be the environment
	// variable name.
	// paramsConfigMap is any additional implementation-defined key/value map to
	// pass to the ValueProvider to help retrieve the value. For example for a
	// ValueProvider that retrieves values from an HTTP source the paramsConfigMap
	// may contain HTTP URL query parameters.
	RetrieveValue(ctx context.Context, onChange func(*ChangeEvent), selector string, paramsConfigMap *config.Map) (RetrievedValue, error)
}

// RetrievedValue holds the result of a call to the RetrieveValue method of a ValueProvider object.
//
// The typical usage is the following:
//
//		r := valueProvider.RetrieveValue(selector, params)
//		r.Get()
//		// wait for onChange() to be called.
//		r.Close()
//		r = valueProvider.RetrieveValue(selector, params)
//		r.Get()
//		// wait for onChange() to be called.
//		r.Close()
//		// repeat RetrieveValue/Get/wait/Close cycle until it is time to shut down the Collector process.
//		// ...
//		mapProvider.Shutdown()
type RetrievedValue interface {
	// Get returns the config value, which should be either a primitive value,
	// such as string or number or a *config.Map.
	// If Close is called before Get or concurrently with Get then Get
	// should return immediately with ErrSessionClosed error.
	// Should never be called concurrently with itself.
	// If ctx is cancelled should return immediately with an error.
	// TODO: use a more specific type for return value instead of interface{}.
	Get(ctx context.Context) (interface{}, error)

	// Close signals that the configuration for which it was used to retrieve values is
	// no longer in use and the object should close and release any watchers that it
	// may have created.
	//
	// This method must be called when the service ends, either in case of success or error.
	//
	// Should never be called concurrentdefaultconfigprovidery with itself.
	// May be called before, after or concurrently with Get.
	// If ctx is cancelled should return immediately with an error.
	//
	// Calling Close on an already closed object should have no effect and should return nil.
	Close(ctx context.Context) error
}
