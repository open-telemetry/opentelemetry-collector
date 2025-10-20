// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package maplist_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	Headers *maplist.MapList[string] `mapstructure:"headers"`
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
	require.EqualError(t, xconfmap.Validate(&tc), `headers: duplicate keys in map-style list: [foo]`)
}

func TestMapListNew(t *testing.T) {
	assert.Equal(t, new(maplist.MapList[int]), maplist.New[int]())

	ml := maplist.WithCapacity[string](4)
	require.NotNil(t, ml)
	assert.Equal(t, 4, cap(*ml))

	assert.Equal(t, &maplist.MapList[int]{
		{"a", 1}, {"b", 2},
	}, maplist.FromMap(map[string]int{
		"a": 1,
		"b": 2,
	}))
}

func TestMapListMethods(t *testing.T) {
	ml := &maplist.MapList[int]{{"a", 1}, {"b", 2}, {"c", 3}}

	type pair = struct {
		k string
		v int
	}
	var kvs []pair
	for k, v := range ml.Iter {
		kvs = append(kvs, pair{k, v})
		if k == "b" {
			break
		}
	}
	assert.Equal(t, []pair{{"a", 1}, {"b", 2}}, kvs)

	v, ok := ml.TryGet("a")
	assert.True(t, ok)
	if ok {
		assert.Equal(t, 1, v)
		assert.Equal(t, 1, ml.Get("a"))
	}
	v, ok = ml.TryGet("d")
	assert.False(t, ok)
	assert.Zero(t, v)
	assert.Zero(t, ml.Get("d"))

	ml.Set("d", 4)
	assert.Equal(t, 4, ml.Len())
	assert.Equal(t, map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}, ml.ToMap())

	ml.Set("d", 5)
	assert.Equal(t, 4, ml.Len())
	assert.Equal(t, map[string]int{"a": 1, "b": 2, "c": 3, "d": 5}, ml.ToMap())
}

func TestMapListNil(t *testing.T) {
	var ml *maplist.MapList[int]

	require.NoError(t, ml.Validate())

	called := false
	for range ml.Iter {
		called = true
	}
	assert.False(t, called)

	v, ok := ml.TryGet("a")
	assert.False(t, ok)
	assert.Zero(t, v)
	assert.Zero(t, ml.Get("a"))

	assert.Panics(t, func() {
		ml.Set("a", 0)
	})

	assert.Zero(t, ml.Len())
	assert.Nil(t, ml.ToMap())
}
