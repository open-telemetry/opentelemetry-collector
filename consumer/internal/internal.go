// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/consumer/internal"

import (
	"errors"
)

var ErrNilFunc = errors.New("nil consumer func")

type BaseImpl struct {
	capabilities Capabilities
}

type BaseConsumer interface {
	Capabilities() Capabilities
}

var _ BaseConsumer = (*BaseImpl)(nil)

type Option interface {
	apply(*BaseImpl)
}

var _ Option = optionFunc(nil)

type optionFunc func(*BaseImpl)

func (f optionFunc) apply(o *BaseImpl) {
	f(o)
}

// Capabilities returns the capabilities of the component
func (bs BaseImpl) Capabilities() Capabilities {
	return bs.capabilities
}

func NewBaseImpl(options ...Option) *BaseImpl {
	bs := &BaseImpl{
		capabilities: Capabilities{MutatesData: false},
	}

	for _, op := range options {
		op.apply(bs)
	}

	return bs
}
