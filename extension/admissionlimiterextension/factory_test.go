// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package admissionlimiterextension

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))

	assert.Equal(t, errLimitOutOfRange, cfg.(*Config).Validate())
}
