// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp

import (
	"testing"

	"go.opentelemetry.io/collector/pdata/internal"
)

func TestMigrateProfiles(_ *testing.T) {
	rps := []*internal.ResourceProfiles{
		{},
	}

	MigrateProfiles(rps)
}
