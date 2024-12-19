// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp

import (
	"testing"

	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"
)

func TestMigrateProfiles(_ *testing.T) {
	rps := []*otlpprofiles.ResourceProfiles{
		{},
	}

	MigrateProfiles(rps)
}
