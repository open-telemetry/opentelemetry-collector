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

// Get returns the config Map. Should never be called after Close.
// Should never be called concurrently with itself or Close.
//
// Deprecated: Not needed, will be removed soon.
func (r Retrieved) Get(ctx context.Context) (*config.Map, error) {
	return r.Map, nil
}

// Close signals that the configuration for which it was used to retrieve values is
// no longer in use and should close and release any watchers that it may have created.
//
// Should block until all resources are closed, and guarantee that `onChange` is not
// going to be called after it returns except when `ctx` is cancelled.
//
// Should never be called concurrently with itself or Get.
//
// Deprecated: Not needed, will be removed soon.
func (r Retrieved) Close(ctx context.Context) error {
	if r.CloseFunc == nil {
		return nil
	}
	return r.CloseFunc(ctx)
}

// GetFunc specifies the function invoked when the Retrieved.Get is being called.
// Deprecated: Not needed, will be removed soon.
type GetFunc func(context.Context) (*config.Map, error)

// Get implements the Retrieved.Get.
// Deprecated: Not needed, will be removed soon.
func (f GetFunc) Get(ctx context.Context) (*config.Map, error) {
	return f(ctx)
}

// Close implements the Retrieved.Close.
// Deprecated: Not needed, will be removed soon.
func (f CloseFunc) Close(ctx context.Context) error {
	if f == nil {
		return nil
	}
	return f(ctx)
}

// RetrievedOption represents the possible options for NewRetrieved.
// Deprecated: Not needed, will be removed soon.
type RetrievedOption func(*Retrieved)

// WithClose overrides the default `Close` function for a Retrieved.
// The default always returns nil.
// Deprecated: Not needed, will be removed soon.
func WithClose(closeFunc CloseFunc) RetrievedOption {
	return func(o *Retrieved) {
		o.CloseFunc = closeFunc
	}
}

// NewRetrieved returns a Retrieved configured with the provided options.
// Deprecated: Not needed, will be removed soon.
func NewRetrieved(getFunc GetFunc, options ...RetrievedOption) (Retrieved, error) {
	if getFunc == nil {
		return Retrieved{}, errors.New("nil getFunc")
	}
	cfg, err := getFunc(context.Background())
	if err != nil {
		return Retrieved{}, err
	}
	ret := Retrieved{
		Map: cfg,
	}
	for _, op := range options {
		op(&ret)
	}
	return ret, nil
}
