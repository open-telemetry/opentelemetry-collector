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
		"idle_conn_timeout":   90 * time.Second,
		"max_idle_conns":      100,
		"force_attempt_http2": true,
	}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(confighttp.NewDefaultCORSConfig()))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(confighttp.NewDefaultServerConfig()))
	assert.Equal(t, map[string]any{
		"cors":                nil,
		"idle_timeout":        60 * time.Second,
		"keep_alives_enabled": true,
		"read_header_timeout": 60 * time.Second,
		"tls":                 nil,
		"write_timeout":       30 * time.Second,
	}, conf.ToStringMap())

	conf = confmap.New()
	require.NoError(t, conf.Marshal(confighttp.AuthConfig{}))
	assert.Equal(t, map[string]any{}, conf.ToStringMap())
}
