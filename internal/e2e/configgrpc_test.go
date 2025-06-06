// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/confmap"
)

func TestConfmapMarshalConfigGRPC(t *testing.T) {
	conf := confmap.New()
	require.NoError(t, conf.Marshal(configgrpc.NewDefaultClientConfig()))
	assert.Equal(t, map[string]any{
		"balancer_name": "round_robin",
	}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(configgrpc.NewDefaultKeepaliveClientConfig()))

	conf = confmap.New()
	require.NoError(t, conf.Marshal(configgrpc.NewDefaultKeepaliveEnforcementPolicy()))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(configgrpc.NewDefaultKeepaliveServerConfig()))

	conf = confmap.New()
	require.NoError(t, conf.Marshal(configgrpc.NewDefaultKeepaliveServerParameters()))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(configgrpc.NewDefaultServerConfig()))
}
