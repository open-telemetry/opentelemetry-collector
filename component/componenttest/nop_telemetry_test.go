// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

func TestNewNopTelemetrySettings(t *testing.T) {
	nts := NewNopTelemetrySettings()
	assert.NotNil(t, nts.Logger)
	assert.NotNil(t, nts.TracerProvider)
	assert.NotPanics(t, func() {
		nts.TracerProvider.Tracer("test")
	})
	assert.NotNil(t, nts.MeterProvider)
	assert.NotPanics(t, func() {
		nts.MeterProvider.Meter("test")
	})
	assert.Equal(t, configtelemetry.LevelNone, nts.MetricsLevel)
	assert.Equal(t, nts.Resource.Attributes().Len(), 0)
}
