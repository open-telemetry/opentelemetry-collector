// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
)

func TestDeprecatedScopeSpans(t *testing.T) {
	ss := new(otlptrace.ScopeSpans)
	rss := []*otlptrace.ResourceSpans{
		{
			ScopeSpans:           []*otlptrace.ScopeSpans{ss},
			DeprecatedScopeSpans: []*otlptrace.ScopeSpans{ss},
		},
		{
			ScopeSpans:           []*otlptrace.ScopeSpans{},
			DeprecatedScopeSpans: []*otlptrace.ScopeSpans{ss},
		},
	}

	MigrateTraces(rss)
	assert.Same(t, ss, rss[0].ScopeSpans[0])
	assert.Same(t, ss, rss[1].ScopeSpans[0])
	assert.Nil(t, rss[0].DeprecatedScopeSpans)
	assert.Nil(t, rss[0].DeprecatedScopeSpans)
}
