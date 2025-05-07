// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
)

func TestDeprecatedScopeLogs(t *testing.T) {
	sl := new(otlplogs.ScopeLogs)
	rls := []*otlplogs.ResourceLogs{
		{
			ScopeLogs:           []*otlplogs.ScopeLogs{sl},
			DeprecatedScopeLogs: []*otlplogs.ScopeLogs{sl},
		},
		{
			ScopeLogs:           []*otlplogs.ScopeLogs{},
			DeprecatedScopeLogs: []*otlplogs.ScopeLogs{sl},
		},
	}

	MigrateLogs(rls)
	assert.Same(t, sl, rls[0].ScopeLogs[0])
	assert.Same(t, sl, rls[1].ScopeLogs[0])
	assert.Nil(t, rls[0].DeprecatedScopeLogs)
	assert.Nil(t, rls[0].DeprecatedScopeLogs)
}
