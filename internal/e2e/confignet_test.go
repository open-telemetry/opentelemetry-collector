// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/confmap"
)

func TestConfmapMarshalConfigNet(t *testing.T) {
	conf := confmap.New()
	require.NoError(t, conf.Marshal(confignet.NewDefaultDialerConfig()))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(confignet.NewDefaultAddrConfig()))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(confignet.NewDefaultTCPAddrConfig()))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())
}
