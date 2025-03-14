// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultBuildInfo(t *testing.T) {
	buildInfo := NewDefaultBuildInfo()
	assert.Equal(t, "otelcol", buildInfo.Command)
	assert.Equal(t, "opentelemetry", buildInfo.Namespace)
	assert.Equal(t, "OpenTelemetry Collector", buildInfo.Description)
	assert.Equal(t, "latest", buildInfo.Version)
}
