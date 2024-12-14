// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Deprecated: [0.116.0] This package is deprecated. Use "go.opentelemetry.io/collector/consumer/xconsumer" instead.
package consumerprofiles // import "go.opentelemetry.io/collector/consumer/consumerprofiles"

import (
	"errors"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
)

var errNilFunc = errors.New("nil consumer func")

// Profiles is an interface that receives pprofile.Profiles, processes it
// as needed, and sends it to the next processing node if any or to the destination.
// Deprecated: [0.116.0] Use xconsumer.Profiles instead.
type Profiles = xconsumer.Profiles

// ConsumeProfilesFunc is a helper function that is similar to ConsumeProfiles.
// Deprecated: [0.116.0] Use xconsumer.ConsumeProfilesFunc instead.
type ConsumeProfilesFunc = xconsumer.ConsumeProfilesFunc

// NewProfiles returns a Profiles configured with the provided options.
// Deprecated: [0.116.0] Use xconsumer.NewProfiles instead.
func NewProfiles(consume ConsumeProfilesFunc, options ...consumer.Option) (Profiles, error) {
	return xconsumer.NewProfiles(consume, options...)
}
