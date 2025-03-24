// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap"
)

func TestConfmapMarshalConfigHTTP(t *testing.T) {
	conf := confmap.New()
	require.NoError(t, conf.Marshal(confighttp.NewDefaultClientConfig()))
	assert.Equal(t, map[string]any{
		"headers":           map[string]any{},
		"idle_conn_timeout": 90 * time.Second,
		"max_idle_conns":    100,
	}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(confighttp.NewDefaultCORSConfig()))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(confighttp.NewDefaultServerConfig()))
	assert.Equal(t, map[string]any{
		"cors":                map[string]any{},
		"idle_timeout":        60 * time.Second,
		"read_header_timeout": 60 * time.Second,
		"response_headers":    map[string]any{},
		"tls":                 map[string]any{},
		"write_timeout":       30 * time.Second,
	}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(confighttp.AuthConfig{}))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())
}
