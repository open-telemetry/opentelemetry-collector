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
// Deprecated: [v0.45.0] Not needed, will be removed soon. You have access to the struct members.
func (r Retrieved) Get(ctx context.Context) (*config.Map, error) {
	return r.Map, nil
}

// GetFunc specifies the function invoked when the Retrieved.Get is being called.
// Deprecated: [v0.45.0] Not needed, will be removed soon. You have access to the struct members.
type GetFunc func(context.Context) (*config.Map, error)

// Get implements the Retrieved.Get.
// Deprecated: [v0.45.0] Not needed, will be removed soon. You have access to the struct members.
func (f GetFunc) Get(ctx context.Context) (*config.Map, error) {
	return f(ctx)
}

// RetrievedOption represents the possible options for NewRetrieved.
// Deprecated: [v0.45.0] Not needed, will be removed soon. You have access to the struct members.
type RetrievedOption func(*Retrieved)

// WithClose overrides the default `Close` function for a Retrieved.
// The default always returns nil.
// Deprecated: [v0.45.0] Not needed, will be removed soon. You have access to the struct members.
func WithClose(closeFunc CloseFunc) RetrievedOption {
	return func(o *Retrieved) {
		o.CloseFunc = closeFunc
	}
}

// NewRetrieved returns a Retrieved configured with the provided options.
// Deprecated: [v0.45.0] Not needed, will be removed soon. You have access to the struct members.
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
