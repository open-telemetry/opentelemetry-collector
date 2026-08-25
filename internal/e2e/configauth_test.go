// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/confmap"
)

func TestConfmapMarshalConfigAuth(t *testing.T) {
	conf := confmap.New()
	require.NoError(t, conf.Marshal(configauth.Config{}))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())
}
