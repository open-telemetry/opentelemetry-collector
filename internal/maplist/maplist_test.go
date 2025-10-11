// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package maplist_test

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/internal/maplist"
)

const headersList = `
headers:
- name: "foo"
  value: "bar"
`

const headersMap = `
headers:
  "foo": "bar"
`

type testConfig struct {
	Headers maplist.MapList[string] `mapstructure:"headers"`
}

func TestMapList(t *testing.T) {
	retrieved1, err := confmap.NewRetrievedFromYAML([]byte(headersList))
	require.NoError(t, err)
	conf1, err := retrieved1.AsConf()
	require.NoError(t, err)
	var tc1 testConfig
	require.NoError(t, conf1.Unmarshal(&tc1))
	assert.NoError(t, xconfmap.Validate(&tc1))

	retrieved2, err := confmap.NewRetrievedFromYAML([]byte(headersMap))
	require.NoError(t, err)
	conf2, err := retrieved2.AsConf()
	require.NoError(t, err)
	var tc2 testConfig
	require.NoError(t, conf2.Unmarshal(&tc2))
	assert.NoError(t, xconfmap.Validate(&tc2))

	assert.Equal(t, tc1, tc2)
}
