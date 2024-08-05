// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterprofiles // import "go.opentelemetry.io/collector/exporter/exporterprofiles"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/internal"
)

// Profiles is an exporter that can consume profiles.
type Profiles = internal.Profiles

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles.
type CreateProfilesFunc = internal.CreateProfilesFunc

// WithProfiles overrides the default "error not supported" implementation for CreateProfilesExporter and the default "undefined" stability level.
func WithProfiles(createProfiles CreateProfilesFunc, sl component.StabilityLevel) exporter.FactoryOption {
	return internal.WithProfiles(createProfiles, sl)
}
