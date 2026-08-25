// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
)

func TestDeprecatedScopeSpans(t *testing.T) {
	ss := new(internal.ScopeSpans)
	rss := []*internal.ResourceSpans{
		{
			ScopeSpans:           []*internal.ScopeSpans{ss},
			DeprecatedScopeSpans: []*internal.ScopeSpans{ss},
		},
		{
			ScopeSpans:           []*internal.ScopeSpans{},
			DeprecatedScopeSpans: []*internal.ScopeSpans{ss},
		},
	}

	MigrateTraces(rss)
	assert.Same(t, ss, rss[0].ScopeSpans[0])
	assert.Same(t, ss, rss[1].ScopeSpans[0])
	assert.Nil(t, rss[0].DeprecatedScopeSpans)
	assert.Nil(t, rss[0].DeprecatedScopeSpans)
}
