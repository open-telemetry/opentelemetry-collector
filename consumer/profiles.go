// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumer // import "go.opentelemetry.io/collector/consumer"

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pprofile"
)

// Profiles is an interface that receives pprofile.Profiles, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type Profiles interface {
	baseConsumer
	// ConsumeProfiles receives pprofile.Profiles for consumption.
	ConsumeProfiles(ctx context.Context, ld pprofile.Profiles) error
}

// ConsumeProfilesFunc is a helper function that is similar to ConsumeProfiles.
type ConsumeProfilesFunc func(ctx context.Context, ld pprofile.Profiles) error

// ConsumeProfiles calls f(ctx, ld).
func (f ConsumeProfilesFunc) ConsumeProfiles(ctx context.Context, ld pprofile.Profiles) error {
	return f(ctx, ld)
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
