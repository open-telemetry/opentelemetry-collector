// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

func TestConfmapMarshalConfigGRPC(t *testing.T) {
	keepaliveClientConfig := map[string]any{
		"time":    time.Second * 10,
		"timeout": time.Second * 10,
	}
	keepaliveServerConfig := map[string]any{
		"server_parameters":  map[string]any{},
		"enforcement_policy": map[string]any{},
	}

	conf := confmap.New()
	require.NoError(t, conf.Marshal(configgrpc.NewDefaultClientConfig()))
	assert.Equal(t, map[string]any{
		"keepalive":     keepaliveClientConfig,
		"balancer_name": "round_robin",
	}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(configgrpc.NewDefaultKeepaliveClientConfig()))
	assert.Equal(t, keepaliveClientConfig, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(configgrpc.NewDefaultKeepaliveEnforcementPolicy()))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(configgrpc.NewDefaultKeepaliveServerConfig()))
	assert.Equal(t, keepaliveServerConfig, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(configgrpc.NewDefaultKeepaliveServerParameters()))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(configgrpc.NewDefaultServerConfig()))
	assert.Equal(t, map[string]any{
		"keepalive": keepaliveServerConfig,
	}, conf.ToStringMap())
}

func TestValidation(t *testing.T) {
	cc := configgrpc.NewDefaultClientConfig()
	require.NoError(t, xconfmap.Validate(cc))

	sc := configgrpc.NewDefaultServerConfig()
	require.Error(t, xconfmap.Validate(sc))

	sc = configgrpc.NewDefaultServerConfig()
	sc.NetAddr = confignet.NewDefaultAddrConfig()
	sc.NetAddr.Transport = confignet.TransportTypeIP6
	require.NoError(t, xconfmap.Validate(sc))
}
