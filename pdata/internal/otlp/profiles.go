// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp // import "go.opentelemetry.io/collector/pdata/internal/otlp"

import (
	"go.opentelemetry.io/collector/pdata/internal"
)

// MigrateProfiles implements any translation needed due to deprecation in OTLP profiles protocol.
// Any pprofile.Unmarshaler implementation from OTLP (proto/json) MUST call this, and the gRPC Server implementation.
func MigrateProfiles(_ []*internal.ResourceProfiles) {}

// PProfileConvertProfilesToReferences is a function pointer that allows pprofileotlp to call
// convertProfilesToReferences from pprofile without creating a cyclic dependency.
var PProfileConvertProfilesToReferences func(any)

// PProfileResolveProfilesReferences is a function pointer that allows pprofileotlp to call
// resolveProfilesReferences from pprofile without creating a cyclic dependency.
var PProfileResolveProfilesReferences func(any)
