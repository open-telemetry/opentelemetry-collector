// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorprofiles // import "go.opentelemetry.io/collector/processor/processorprofiles"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/internal"
)

// Profiles is a processor that can consume profiles.
type Profiles = internal.Profiles

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles().
type CreateProfilesFunc = internal.CreateProfilesFunc

// WithProfiles overrides the default "error not supported" implementation for CreateProfiles and the default "undefined" stability level.
func WithProfiles(createProfiles CreateProfilesFunc, sl component.StabilityLevel) processor.FactoryOption {
	return internal.WithProfiles(createProfiles, sl)
}
