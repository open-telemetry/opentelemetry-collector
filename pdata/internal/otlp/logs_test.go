// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
)

func TestDeprecatedScopeLogs(t *testing.T) {
	sl := new(internal.ScopeLogs)
	rls := []*internal.ResourceLogs{
		{
			ScopeLogs:           []*internal.ScopeLogs{sl},
			DeprecatedScopeLogs: []*internal.ScopeLogs{sl},
		},
		{
			ScopeLogs:           []*internal.ScopeLogs{},
			DeprecatedScopeLogs: []*internal.ScopeLogs{sl},
		},
	}

	MigrateLogs(rls)
	assert.Same(t, sl, rls[0].ScopeLogs[0])
	assert.Same(t, sl, rls[1].ScopeLogs[0])
	assert.Nil(t, rls[0].DeprecatedScopeLogs)
	assert.Nil(t, rls[0].DeprecatedScopeLogs)
}
