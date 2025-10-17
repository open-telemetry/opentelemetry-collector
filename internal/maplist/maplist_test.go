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

const headersBad = `
headers:
  "bad": 1
`

const headersDupe = `
headers:
- name: "foo"
  value: "bar"
- name: "foo"
  value: "baz"
`

type testConfig struct {
	Headers maplist.MapList[string] `mapstructure:"headers"`
}

func TestMapListDuality(t *testing.T) {
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

func TestMapListUnmarshalError(t *testing.T) {
	retrieved, err := confmap.NewRetrievedFromYAML([]byte(headersBad))
	require.NoError(t, err)
	conf, err := retrieved.AsConf()
	require.NoError(t, err)
	var tc testConfig
	// Not sure if there is a way to change the error message to include the map case?
	assert.EqualError(t, conf.Unmarshal(&tc),
		"decoding failed due to the following error(s):\n\n"+
			"'headers' decoding failed due to the following error(s):\n\n"+
			"'[bad]' expected type 'string', got unconvertible type 'int'")
}

func TestMapListValidate(t *testing.T) {
	retrieved, err := confmap.NewRetrievedFromYAML([]byte(headersDupe))
	require.NoError(t, err)
	conf, err := retrieved.AsConf()
	require.NoError(t, err)
	var tc testConfig
	require.NoError(t, conf.Unmarshal(&tc))
	assert.EqualError(t, xconfmap.Validate(&tc), `headers: duplicate keys in map-style list: [foo]`)
}

func TestMapListFromMap(t *testing.T) {
	assert.Equal(t, maplist.MapList[int]{
		{"a", 1}, {"b", 2},
	}, maplist.FromMap(map[string]int{
		"a": 1,
		"b": 2,
	}))
}

func TestMapListMethods(t *testing.T) {
	ml := maplist.MapList[int]{{"a", 1}, {"b", 2}, {"c", 3}}

	type pair = struct {
		k string
		v int
	}
	var kvs []pair
	for k, v := range ml.Pairs {
		kvs = append(kvs, pair{k, v})
		if k == "b" {
			break
		}
	}
	assert.Equal(t, []pair{{"a", 1}, {"b", 2}}, kvs)

	v, ok := ml.Get("a")
	assert.True(t, ok)
	if ok {
		assert.Equal(t, 1, v)
	}
	v, ok = ml.Get("d")
	assert.False(t, ok)
	assert.Zero(t, v)

	assert.Equal(t, map[string]int{"a": 1, "b": 2, "c": 3}, ml.ToMap())
}
