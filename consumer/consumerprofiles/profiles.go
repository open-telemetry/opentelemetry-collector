// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumerprofiles // import "go.opentelemetry.io/collector/consumer/consumerprofiles"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/internal"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var errNilFunc = errors.New("nil consumer func")

// Profiles is an interface that receives pprofile.Profiles, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type Profiles interface {
	internal.BaseConsumer
	// ConsumeProfiles receives pprofile.Profiles for consumption.
	ConsumeProfiles(ctx context.Context, td pprofile.Profiles) error
}

// ConsumeProfilesFunc is a helper function that is similar to ConsumeProfiles.
type ConsumeProfilesFunc func(ctx context.Context, td pprofile.Profiles) error

// ConsumeProfiles calls f(ctx, td).
func (f ConsumeProfilesFunc) ConsumeProfiles(ctx context.Context, td pprofile.Profiles) error {
	return f(ctx, td)
}

type baseProfiles struct {
	*internal.BaseImpl
	ConsumeProfilesFunc
}

// NewProfiles returns a Profiles configured with the provided options.
func NewProfiles(consume ConsumeProfilesFunc, options ...consumer.Option) (Profiles, error) {
	if consume == nil {
		return nil, errNilFunc
	}
	return &baseProfiles{
		BaseImpl:            internal.NewBaseImpl(options...),
		ConsumeProfilesFunc: consume,
	}, nil
}
