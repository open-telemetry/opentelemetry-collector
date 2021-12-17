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

package configmapprovider // import "go.opentelemetry.io/collector/config/configmapprovider"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/config"
)

// Retrieved holds the result of a call to the Retrieve method of a Provider object.
// This interface cannot be directly implemented. Implementations must use the NewRetrieved helper.
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

	// privateRetrieved is an unexported func to disallow direct implementation.
	privateRetrieved()
}

// GetFunc specifies the function invoked when the Retrieved.Get is being called.
type GetFunc func(context.Context) (*config.Map, error)

// Get implements the Retrieved.Get.
func (f GetFunc) Get(ctx context.Context) (*config.Map, error) {
	return f(ctx)
}

// CloseFunc specifies the function invoked when the Retrieved.Close is being called.
type CloseFunc func(context.Context) error

// Close implements the Retrieved.Close.
func (f CloseFunc) Close(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}

// RetrievedOption represents the possible options for NewRetrieved.
type RetrievedOption func(*retrieved)

// WithClose overrides the default `Close` function for a Retrieved.
// The default always returns nil.
func WithClose(closeFunc CloseFunc) RetrievedOption {
	return func(o *retrieved) {
		o.CloseFunc = closeFunc
	}
}

type retrieved struct {
	GetFunc
	CloseFunc
}

func (retrieved) privateRetrieved() {}

// NewRetrieved returns a Retrieved configured with the provided options.
func NewRetrieved(getFunc GetFunc, options ...RetrievedOption) (Retrieved, error) {
	if getFunc == nil {
		return nil, errors.New("nil getFunc")
	}
	ret := &retrieved{
		GetFunc: getFunc,
	}
	for _, op := range options {
		op(ret)
	}
	return ret, nil
}
