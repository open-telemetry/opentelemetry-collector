// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricData(t *testing.T) {
	for _, arg := range []struct {
		metricData    MetricData
		typ           string
		hasAggregated bool
		hasMonotonic  bool
	}{
		{&gauge{}, "Gauge", false, false},
		{&sum{}, "Sum", true, true},
	} {
		assert.Equal(t, arg.typ, arg.metricData.Type())
		assert.Equal(t, arg.hasAggregated, arg.metricData.HasAggregated())
		assert.Equal(t, arg.hasMonotonic, arg.metricData.HasMonotonic())
	}
}
