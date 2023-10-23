// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp // import "go.opentelemetry.io/collector/pdata/internal/otlp"

import (
	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1"
)

// MigrateProfiles implements any translation needed due to deprecation in OTLP logs protocol.
// Any plog.Unmarshaler implementation from OTLP (proto/json) MUST call this, and the gRPC Server implementation.
func MigrateProfiles(rls []*otlpprofiles.ResourceProfiles) {
	for _, rl := range rls {
		if len(rl.ScopeProfiles) == 0 {
			rl.ScopeProfiles = rl.DeprecatedScopeProfiles
		}
		rl.DeprecatedScopeProfiles = nil
	}
}
