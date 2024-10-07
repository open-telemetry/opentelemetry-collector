// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtimeout

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalTextValid(t *testing.T) {
	for _, valid := range []string{
		string(PolicySustain),
		string(PolicyAbort),
		string(PolicyIgnore),
		string(policyUnset),
		string(PolicyDefault),
	} {
		t.Run(valid, func(t *testing.T) {
			pol := Policy(valid)
			require.NoError(t, pol.Validate())
			assert.Equal(t, Policy(valid), pol)
		})
	}
}

func TestUnmarshalTextInvalid(t *testing.T) {
	for _, invalid := range []string{
		"unknown",
		"invalid",
		"etc",
	} {
		t.Run(invalid, func(t *testing.T) {
			pol := Policy(invalid)
			require.Error(t, pol.Validate())
			assert.Equal(t, Policy(invalid), pol)
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "default.yaml"))
	require.NoError(t, err)
	cfg := NewDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	assert.Equal(t,
		TimeoutConfig{
			Timeout:            5 * time.Second,
			ShortTimeoutPolicy: "",
		}, cfg)
}

func TestDefaulPolicy(t *testing.T) {
	// The legacy behavior of the exporterhelper is to sustain a short timeout.
	assert.Equal(t, PolicySustain, PolicyDefault)
}

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	cfg := NewDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	assert.Equal(t,
		TimeoutConfig{
			Timeout:            17 * time.Second,
			ShortTimeoutPolicy: "abort",
		}, cfg)
}

func TestUnmarshalInvalidConfig(t *testing.T) {
	type test struct {
		file   string
		expect string
	}
	for _, tt := range []test{
		{
			file:   "invalid_timeout.yaml",
			expect: "'timeout' must be non-negative",
		},
		{
			file:   "invalid_policy.yaml",
			expect: "unsupported 'short_timeout_policy' hope",
		},
	} {
		t.Run(tt.file, func(t *testing.T) {
			cm, err := confmaptest.LoadConf(filepath.Join("testdata", tt.file))
			require.NoError(t, err)
			cfg := NewDefaultConfig()
			assert.NoError(t, cm.Unmarshal(&cfg))
			assert.ErrorContains(t, component.ValidateConfig(cfg), tt.expect)
		})
	}
}
