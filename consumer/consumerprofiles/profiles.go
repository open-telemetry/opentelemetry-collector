// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Deprecated: [0.115.0] This package is deprecated. Use "go.opentelemetry.io/collector/consumer/consumerexp" instead.
package consumerprofiles // import "go.opentelemetry.io/collector/consumer/consumerprofiles"

import (
	"errors"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerexp"
)

var errNilFunc = errors.New("nil consumer func")

// Profiles is an interface that receives pprofile.Profiles, processes it
// as needed, and sends it to the next processing node if any or to the destination.
// Deprecated: [0.115.0] Use consumerexp.Profiles instead.
type Profiles = consumerexp.Profiles

// ConsumeProfilesFunc is a helper function that is similar to ConsumeProfiles.
// Deprecated: [0.115.0] Use consumerexp.ConsumeProfilesFunc instead.
type ConsumeProfilesFunc = consumerexp.ConsumeProfilesFunc

// NewProfiles returns a Profiles configured with the provided options.
// Deprecated: [0.115.0] Use consumerexp.NewProfiles instead.
func NewProfiles(consume ConsumeProfilesFunc, options ...consumer.Option) (Profiles, error) {
	return consumerexp.NewProfiles(consume, options...)
}
