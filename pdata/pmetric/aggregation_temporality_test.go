// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAggregationTemporalityString(t *testing.T) {
	assert.Equal(t, "Unspecified", AggregationTemporalityUnspecified.String())
	assert.Equal(t, "Delta", AggregationTemporalityDelta.String())
	assert.Equal(t, "Cumulative", AggregationTemporalityCumulative.String())
	assert.Empty(t, (AggregationTemporalityCumulative + 1).String())
}
