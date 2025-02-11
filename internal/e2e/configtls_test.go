// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap"
)

func TestConfmapMarshalConfigTLS(t *testing.T) {
	conf := confmap.New()
	require.NoError(t, conf.Marshal(configtls.NewDefaultConfig()))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(configtls.NewDefaultClientConfig()))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(configtls.NewDefaultServerConfig()))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())
}
