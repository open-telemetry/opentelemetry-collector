// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

func TestDeprecatedScopeMetrics(t *testing.T) {
	sm := new(otlpmetrics.ScopeMetrics)
	rms := []*otlpmetrics.ResourceMetrics{
		{
			ScopeMetrics:           []*otlpmetrics.ScopeMetrics{sm},
			DeprecatedScopeMetrics: []*otlpmetrics.ScopeMetrics{sm},
		},
		{
			ScopeMetrics:           []*otlpmetrics.ScopeMetrics{},
			DeprecatedScopeMetrics: []*otlpmetrics.ScopeMetrics{sm},
		},
	}

	MigrateMetrics(rms)
	assert.Same(t, sm, rms[0].ScopeMetrics[0])
	assert.Same(t, sm, rms[1].ScopeMetrics[0])
	assert.Nil(t, rms[0].DeprecatedScopeMetrics)
	assert.Nil(t, rms[0].DeprecatedScopeMetrics)
}
