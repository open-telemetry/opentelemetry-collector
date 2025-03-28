// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricTypeString(t *testing.T) {
	assert.Equal(t, "Empty", MetricTypeEmpty.String())
	assert.Equal(t, "Gauge", MetricTypeGauge.String())
	assert.Equal(t, "Sum", MetricTypeSum.String())
	assert.Equal(t, "Histogram", MetricTypeHistogram.String())
	assert.Equal(t, "ExponentialHistogram", MetricTypeExponentialHistogram.String())
	assert.Equal(t, "Summary", MetricTypeSummary.String())
	assert.Empty(t, (MetricTypeSummary + 1).String())
}
