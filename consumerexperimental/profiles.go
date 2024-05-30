// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumerexperimental // import "go.opentelemetry.io/collector/consumerexperimental"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

type baseConsumer interface {
	Capabilities() consumer.Capabilities
}

var errNilFunc = errors.New("nil consumer func")

// Profiles is an interface that receives pprofile.Profiles, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type Profiles interface {
	baseConsumer
	// ConsumeProfiles receives pprofile.Profiles for consumption.
	ConsumeProfiles(ctx context.Context, td pprofile.Profiles) error
}

// ConsumeProfilesFunc is a helper function that is similar to ConsumeProfiles.
type ConsumeProfilesFunc func(ctx context.Context, td pprofile.Profiles) error

// ConsumeProfiles calls f(ctx, td).
func (f ConsumeProfilesFunc) ConsumeProfiles(ctx context.Context, td pprofile.Profiles) error {
	return f(ctx, td)
}

type baseImpl struct {
	capabilities consumer.Capabilities
}

// Capabilities returns the capabilities of the component
func (bs baseImpl) Capabilities() consumer.Capabilities {
	return bs.capabilities
}

// Option to construct new consumers.
type Option func(*baseImpl)

// WithCapabilities overrides the default GetCapabilities function for a processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities consumer.Capabilities) Option {
	return func(o *baseImpl) {
		o.capabilities = capabilities
	}
}

type baseProfiles struct {
	*baseImpl
	ConsumeProfilesFunc
}

// NewProfiles returns a Profiles configured with the provided options.
func NewProfiles(consume ConsumeProfilesFunc, options ...Option) (Profiles, error) {
	if consume == nil {
		return nil, errNilFunc
	}
	return &baseProfiles{
		baseImpl:            newBaseImpl(options...),
		ConsumeProfilesFunc: consume,
	}, nil
}

func newBaseImpl(options ...Option) *baseImpl {
	bs := &baseImpl{
		capabilities: consumer.Capabilities{MutatesData: false},
	}

	for _, op := range options {
		op(bs)
	}

	return bs
}
