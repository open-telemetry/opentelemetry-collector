// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofileotlp // import "go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"

import "go.opentelemetry.io/collector/pdata/internal/otlp"

// DevelopmentVersion is the OTLP Profiles development version encoded and decoded
// by this package. It is determined by the compiled-in schema, not configuration.
const DevelopmentVersion = otlp.ProfilesDevelopmentVersion

// DevelopmentVersionHeader is the request metadata key for the OTLP Profiles
// development version. HTTP header names are case-insensitive.
const DevelopmentVersionHeader = otlp.ProfilesDevelopmentVersionHeader

// ValidateDevelopmentVersion validates all values of the Profiles development
// version request metadata. It must be called before deserializing the payload.
// Absent metadata identifies version 1. Malformed, repeated, and unsupported
// values are rejected. The gRPC server performs this validation automatically.
func ValidateDevelopmentVersion(values []string) error {
	return otlp.ValidateProfilesDevelopmentVersion(values)
}
