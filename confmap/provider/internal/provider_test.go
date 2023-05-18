// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
)

func TestNewRetrievedFromYAML(t *testing.T) {
	ret, err := NewRetrievedFromYAML([]byte{})
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, confmap.New(), retMap)
	assert.NoError(t, ret.Close(context.Background()))
}

func TestNewRetrievedFromYAMLWithOptions(t *testing.T) {
	want := errors.New("my error")
	ret, err := NewRetrievedFromYAML([]byte{}, confmap.WithRetrievedClose(func(context.Context) error { return want }))
	require.NoError(t, err)
	retMap, err := ret.AsConf()
	require.NoError(t, err)
	assert.Equal(t, confmap.New(), retMap)
	assert.Equal(t, want, ret.Close(context.Background()))
}

func TestNewRetrievedFromYAMLInvalidYAMLBytes(t *testing.T) {
	_, err := NewRetrievedFromYAML([]byte("[invalid:,"))
	assert.Error(t, err)
}

func TestNewRetrievedFromYAMLInvalidAsMap(t *testing.T) {
	ret, err := NewRetrievedFromYAML([]byte("string"))
	require.NoError(t, err)

	_, err = ret.AsConf()
	assert.Error(t, err)
}
