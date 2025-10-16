// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntitySlice(t *testing.T) {
	es := NewEntitySlice()
	assert.Equal(t, 0, es.Len())

	e1, err := es.AppendEmpty("service")
	require.NoError(t, err)
	assert.Equal(t, 1, es.Len())
	assert.Equal(t, "service", e1.Type())

	e2, err := es.AppendEmpty("host")
	require.NoError(t, err)
	assert.Equal(t, 2, es.Len())
	assert.Equal(t, "host", e2.Type())
}

func TestEntitySlice_AppendEmpty(t *testing.T) {
	es := NewEntitySlice()
	e, err := es.AppendEmpty("service")
	require.NoError(t, err)
	assert.Equal(t, "service", es.At(0).Type())
	assert.Equal(t, "service", e.Type())
}

func TestEntitySlice_At(t *testing.T) {
	es := NewEntitySlice()
	e1, err := es.AppendEmpty("service")
	require.NoError(t, err)
	e1.SetSchemaURL("https://opentelemetry.io/schemas/1.0.0")

	_, err = es.AppendEmpty("host")
	require.NoError(t, err)

	assert.Equal(t, "service", es.At(0).Type())
	assert.Equal(t, "https://opentelemetry.io/schemas/1.0.0", es.At(0).SchemaURL())
	assert.Equal(t, "host", es.At(1).Type())
	assert.Empty(t, es.At(1).SchemaURL())
}

func TestEntitySlice_EnsureCapacity(t *testing.T) {
	es := NewEntitySlice()
	es.EnsureCapacity(5)
	for i := 0; i < 5; i++ {
		_, err := es.AppendEmpty(fmt.Sprintf("type%d", i))
		require.NoError(t, err)
	}
	assert.Equal(t, 5, es.Len())

	es.EnsureCapacity(3)
	assert.Equal(t, 5, es.Len())

	es.EnsureCapacity(8)
	for i := 5; i < 8; i++ {
		_, err := es.AppendEmpty(fmt.Sprintf("type%d", i))
		require.NoError(t, err)
	}
	assert.Equal(t, 8, es.Len())
}

func TestEntitySlice_SharedAttributes(t *testing.T) {
	es := NewEntitySlice()

	e1, err := es.AppendEmpty("service")
	require.NoError(t, err)
	err = e1.IDAttributes().PutStr("service.name", "my-service")
	require.NoError(t, err)

	e2, err := es.AppendEmpty("host")
	require.NoError(t, err)
	err = e2.IDAttributes().PutStr("host.name", "my-host")
	require.NoError(t, err)

	val, ok := es.At(0).IDAttributes().Get("service.name")
	assert.True(t, ok)
	assert.Equal(t, "my-service", val.Str())

	val, ok = es.At(1).IDAttributes().Get("host.name")
	assert.True(t, ok)
	assert.Equal(t, "my-host", val.Str())

	_, ok = es.At(0).IDAttributes().Get("host.name")
	assert.False(t, ok)

	_, ok = es.At(1).IDAttributes().Get("service.name")
	assert.False(t, ok)
}

func TestEntitySlice_AppendEmpty_EmptyType(t *testing.T) {
	es := NewEntitySlice()
	_, err := es.AppendEmpty("")
	require.ErrorIs(t, err, ErrEmptyEntityType)
	assert.Equal(t, 0, es.Len())
}

func TestEntitySlice_AppendEmpty_DuplicateType(t *testing.T) {
	es := NewEntitySlice()
	_, err := es.AppendEmpty("service")
	require.NoError(t, err)
	assert.Equal(t, 1, es.Len())

	_, err = es.AppendEmpty("service")
	require.ErrorIs(t, err, ErrDuplicateEntityType)
	assert.Equal(t, 1, es.Len())
}
