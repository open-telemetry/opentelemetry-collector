// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1"
)

func TestDeprecatedScopeProfiles(t *testing.T) {
	sl := new(otlpprofiles.ScopeProfiles)
	rls := []*otlpprofiles.ResourceProfiles{
		{
			ScopeProfiles:           []*otlpprofiles.ScopeProfiles{sl},
			DeprecatedScopeProfiles: []*otlpprofiles.ScopeProfiles{sl},
		},
		{
			ScopeProfiles:           []*otlpprofiles.ScopeProfiles{},
			DeprecatedScopeProfiles: []*otlpprofiles.ScopeProfiles{sl},
		},
	}

	MigrateProfiles(rls)
	assert.Same(t, sl, rls[0].ScopeProfiles[0])
	assert.Same(t, sl, rls[1].ScopeProfiles[0])
	assert.Nil(t, rls[0].DeprecatedScopeProfiles)
	assert.Nil(t, rls[0].DeprecatedScopeProfiles)
}
