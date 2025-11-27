// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configopaque_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

const headersList = `
headers:
- name: "a"
  value: "b"
- name: "c"
  value: "d"
`

const headersMap = `
headers:
  "a": "b"
  "c": "d"
`

const headersBad1 = `
headers:
  "bad": 1
`

const headersBad2 = `
headers: "foo"
`

const headersDupe = `
headers:
- name: "foo"
  value: "bar"
- name: "foo"
  value: "baz"
`

type testConfig struct {
	Headers configopaque.MapList `mapstructure:"headers"`
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
	var tc testConfig

	retrieved, err := confmap.NewRetrievedFromYAML([]byte(headersBad1))
	require.NoError(t, err)
	conf, err := retrieved.AsConf()
	require.NoError(t, err)
	require.EqualError(t, conf.Unmarshal(&tc),
		"decoding failed due to the following error(s):\n\n"+
			"'headers' decoding failed due to the following error(s):\n\n"+
			"'[bad]' expected type 'configopaque.String', got unconvertible type 'int'")

	retrieved, err = confmap.NewRetrievedFromYAML([]byte(headersBad2))
	require.NoError(t, err)
	conf, err = retrieved.AsConf()
	require.NoError(t, err)
	// Not sure if there is a way to change the error message to include the map case?
	require.EqualError(t, conf.Unmarshal(&tc),
		"decoding failed due to the following error(s):\n\n"+
			"'headers' source data must be an array or slice, got string")
}

func TestMapListValidate(t *testing.T) {
	retrieved, err := confmap.NewRetrievedFromYAML([]byte(headersDupe))
	require.NoError(t, err)
	conf, err := retrieved.AsConf()
	require.NoError(t, err)
	var tc testConfig
	require.NoError(t, conf.Unmarshal(&tc))
	require.EqualError(t, xconfmap.Validate(&tc), `headers: duplicate keys in map-style list: [foo]`)
}

func TestMapListMethods(t *testing.T) {
	ml := configopaque.MapList{
		{Name: "a", Value: "1"},
		{Name: "b", Value: "2"},
		{Name: "c", Value: "3"},
	}

	type pair = struct {
		k string
		v configopaque.String
	}
	var kvs []pair
	for k, v := range ml.Iter {
		kvs = append(kvs, pair{k, v})
		if k == "b" {
			break
		}
	}
	assert.Equal(t, []pair{{"a", "1"}, {"b", "2"}}, kvs)

	v, ok := ml.Get("a")
	assert.True(t, ok)
	if ok {
		assert.Equal(t, configopaque.String("1"), v)
	}
	v, ok = ml.Get("d")
	assert.False(t, ok)
	assert.Zero(t, v)

	ml2 := ml
	assert.Len(t, ml2, 3)

	// Set existing key
	ml2.Set("c", "4")
	assert.Len(t, ml, 3)
	assert.Len(t, ml2, 3)
	v, _ = ml.Get("c")
	assert.Equal(t, configopaque.String("3"), v)
	v, _ = ml2.Get("c")
	assert.Equal(t, configopaque.String("4"), v)

	// Set new key
	ml2.Set("d", "5")
	assert.Len(t, ml, 3)
	assert.Len(t, ml2, 4)
	_, ok = ml.Get("d")
	assert.False(t, ok)
	v, ok = ml2.Get("d")
	assert.True(t, ok)
	assert.Equal(t, configopaque.String("5"), v)
}

func TestMapListNil(t *testing.T) {
	var ml *configopaque.MapList
	assert.Panics(t, func() {
		ml.Set("a", "0")
	})
}
