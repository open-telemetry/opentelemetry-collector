// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper // import "go.opentelemetry.io/collector/extension/extensionlimiter/limiterhelper"

import (
	"go.opentelemetry.io/collector/extension/extensionlimiter"
)

// WrapperProvider follows the provider pattern for
// the Wrapper type
type WrapperProvider interface {
	GetWrapper(extensionlimiter.WeightKey, ...extensionlimiter.Option) (Wrapper, error)
}

// GetWrapperFunc is an easy way to build GetWrapper functions.
type GetWrapperFunc func(extensionlimiter.WeightKey, ...extensionlimiter.Option) (Wrapper, error)

// GetWrapper implements WrapperProvider.
func (f GetWrapperFunc) GetWrapper(key extensionlimiter.WeightKey, opts ...extensionlimiter.Option) (Wrapper, error) {
	if f == nil {
		return NewWrapperImpl(nil), nil
	}
	return f(key, opts...)
}

type limiterWrapperProviderImpl struct {
	GetWrapperFunc
}

var _ WrapperProvider = limiterWrapperProviderImpl{}

// NewWrapperProviderImpl returns a functional implementation of
// WrapperProvider.  Use a nil argument for the no-op implementation.
func NewWrapperProviderImpl(f GetWrapperFunc) WrapperProvider {
	return limiterWrapperProviderImpl{
		GetWrapperFunc: f,
	}
}
