// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRetrieved(t *testing.T) {
	ret, err := NewRetrieved(nil)
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, New(), retMap)
	assert.NoError(t, ret.Close(context.Background()))
}

func TestNewRetrievedWithOptions(t *testing.T) {
	want := errors.New("my error")
	ret, err := NewRetrieved(nil, WithRetrievedClose(func(context.Context) error { return want }))
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, New(), retMap)
	assert.Equal(t, want, ret.Close(context.Background()))
}

func TestNewRetrievedUnsupportedType(t *testing.T) {
	_, err := NewRetrieved(errors.New("my error"))
	require.Error(t, err)
}

func TestNewRetrievedFromYAML(t *testing.T) {
	ret, err := NewRetrievedFromYAML([]byte{})
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, New(), retMap)
	assert.NoError(t, ret.Close(context.Background()))
}

func TestNewRetrievedFromYAMLWithOptions(t *testing.T) {
	want := errors.New("my error")
	ret, err := NewRetrievedFromYAML([]byte{}, WithRetrievedClose(func(context.Context) error { return want }))
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, New(), retMap)
	assert.Equal(t, want, ret.Close(context.Background()))
}

func TestNewRetrievedFromYAMLInvalidYAMLBytes(t *testing.T) {
	ret, err := NewRetrievedFromYAML([]byte("[invalid:,"))
	require.NoError(t, err)

	_, err = ret.AsConf()
	require.Error(t, err)

	str, err := ret.AsString()
	require.NoError(t, err)
	assert.Equal(t, "[invalid:,", str)

	raw, err := ret.AsRaw()
	require.NoError(t, err)
	assert.Equal(t, "[invalid:,", raw)
}

func TestNewRetrievedFromYAMLInvalidAsMap(t *testing.T) {
	ret, err := NewRetrievedFromYAML([]byte("string"))
	require.NoError(t, err)

	_, err = ret.AsConf()
	require.Error(t, err)

	str, err := ret.AsString()
	require.NoError(t, err)
	assert.Equal(t, "string", str)
}

func TestNewRetrievedFromYAMLString(t *testing.T) {
	tests := []struct {
		yaml       string
		value      any
		altStrRepr string
		strReprErr string
	}{
		{
			yaml:  "string",
			value: "string",
		},
		{
			yaml:       "\"string\"",
			value:      "\"string\"",
			altStrRepr: "\"string\"",
		},
		{
			yaml:  "123",
			value: 123,
		},
		{
			yaml:  "2023-03-20T03:17:55.432328Z",
			value: time.Date(2023, 3, 20, 3, 17, 55, 432328000, time.UTC),
		},
		{
			yaml:  "true",
			value: true,
		},
		{
			yaml:  "0123",
			value: 0o123,
		},
		{
			yaml:  "0x123",
			value: 0x123,
		},
		{
			yaml:  "0b101",
			value: 0b101,
		},
		{
			yaml:  "0.123",
			value: 0.123,
		},
		{
			yaml:  "{key: value}",
			value: map[string]any{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.yaml, func(t *testing.T) {
			ret, err := NewRetrievedFromYAML([]byte(tt.yaml))
			require.NoError(t, err)

			raw, err := ret.AsRaw()
			require.NoError(t, err)
			assert.Equal(t, tt.value, raw)

			str, err := ret.AsString()
			if tt.strReprErr != "" {
				assert.ErrorContains(t, err, tt.strReprErr)
				return
			}
			require.NoError(t, err)

			if tt.altStrRepr != "" {
				assert.Equal(t, tt.altStrRepr, str)
			} else {
				assert.Equal(t, tt.yaml, str)
			}
		})
	}

}
