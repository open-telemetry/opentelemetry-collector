// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func Test_bufferedCore_Level(t *testing.T) {
	bc := newBufferedCore(zapcore.InfoLevel)
	assert.Equal(t, zapcore.InfoLevel, bc.Level())
}

func Test_bufferedCore_Check(t *testing.T) {
	t.Run("check passed", func(t *testing.T) {
		bc := newBufferedCore(zapcore.InfoLevel)
		e := zapcore.Entry{
			Level: zapcore.InfoLevel,
		}
		expected := &zapcore.CheckedEntry{}
		expected = expected.AddCore(e, bc)
		ce := bc.Check(e, nil)
		assert.Equal(t, expected, ce)
	})

	t.Run("check did not pass", func(t *testing.T) {
		bc := newBufferedCore(zapcore.InfoLevel)
		e := zapcore.Entry{
			Level: zapcore.DebugLevel,
		}
		ce := bc.Check(e, nil)
		assert.Nil(t, ce)
	})
}

func Test_bufferedCore_With(t *testing.T) {
	bc := newBufferedCore(zapcore.InfoLevel)
	bc.logsTaken = true
	bc.context = []zapcore.Field{
		{Key: "original", String: "context"},
	}
	inputs := []zapcore.Field{
		{Key: "test", String: "passed"},
	}
	expected := []zapcore.Field{
		{Key: "original", String: "context"},
		{Key: "test", String: "passed"},
	}
	newBC := bc.With(inputs)
	assert.Equal(t, expected, newBC.(*bufferedCore).context)
	assert.True(t, newBC.(*bufferedCore).logsTaken)
}

func Test_bufferedCore_Write(t *testing.T) {
	bc := newBufferedCore(zapcore.InfoLevel)
	e := zapcore.Entry{
		Level:   zapcore.DebugLevel,
		Message: "test",
	}
	fields := []zapcore.Field{
		{Key: "field1", String: "value1"},
	}
	err := bc.Write(e, fields)
	require.NoError(t, err)

	expected := loggedEntry{
		e,
		fields,
	}
	require.Len(t, bc.logs, 1)
	require.Equal(t, expected, bc.logs[0])
}

func Test_bufferedCore_Sync(t *testing.T) {
	bc := newBufferedCore(zapcore.InfoLevel)
	assert.NoError(t, bc.Sync())
}

func Test_bufferedCore_TakeLogs(t *testing.T) {
	bc := newBufferedCore(zapcore.InfoLevel)
	e := zapcore.Entry{
		Level:   zapcore.DebugLevel,
		Message: "test",
	}
	fields := []zapcore.Field{
		{Key: "field1", String: "value1"},
	}
	err := bc.Write(e, fields)
	require.NoError(t, err)

	expected := []loggedEntry{
		{
			e,
			fields,
		},
	}
	assert.Equal(t, expected, bc.TakeLogs())
	assert.Nil(t, bc.logs)

	require.Error(t, bc.Write(e, fields))
	assert.Nil(t, bc.TakeLogs())
}
