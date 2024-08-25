// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverprofiles // import "go.opentelemetry.io/collector/receiver/receiverprofiles"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/internal"
)

// Profiles receiver receives profiles.
// Its purpose is to translate data from any format to the collector's internal profile format.
// ProfilessReceiver feeds a consumerprofiles.Profiles with data.
//
// For example, it could be a pprof data source which translates pprof profiles into pprofile.Profiles.
type Profiles = internal.Profiles

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles.
type CreateProfilesFunc = internal.CreateProfilesFunc

// WithProfiles overrides the default "error not supported" implementation for CreateProfilesReceiver and the default "undefined" stability level.
func WithProfiles(createProfilesReceiver CreateProfilesFunc, sl component.StabilityLevel) receiver.FactoryOption {
	return internal.WithProfiles(createProfilesReceiver, sl)
}
