// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func Test_collectorCore_Enabled(t *testing.T) {
	cc := newCollectorCore(newBufferedCore(zapcore.InfoLevel))
	assert.True(t, cc.Enabled(zapcore.ErrorLevel))
	assert.False(t, cc.Enabled(zapcore.DebugLevel))
}

func Test_collectorCore_Check(t *testing.T) {
	t.Run("check passed", func(t *testing.T) {
		bc := newBufferedCore(zapcore.InfoLevel)
		cc := newCollectorCore(bc)
		e := zapcore.Entry{
			Level: zapcore.InfoLevel,
		}
		expected := &zapcore.CheckedEntry{}
		expected = expected.AddCore(e, bc)
		assert.Equal(t, expected, cc.Check(e, nil))
	})

	t.Run("check did not pass", func(t *testing.T) {
		cc := newCollectorCore(newBufferedCore(zapcore.InfoLevel))
		e := zapcore.Entry{
			Level: zapcore.DebugLevel,
		}
		assert.Nil(t, cc.Check(e, nil))
	})
}

func Test_collectorCore_With(t *testing.T) {
	cc := newCollectorCore(newBufferedCore(zapcore.InfoLevel))
	cc.loadDelegate().(*bufferedCore).context = []zapcore.Field{
		{Key: "original", String: "context"},
	}
	inputs := []zapcore.Field{
		{Key: "test", String: "passed"},
	}
	expected := []zapcore.Field{
		{Key: "original", String: "context"},
		{Key: "test", String: "passed"},
	}
	newCC := cc.With(inputs)
	assert.Equal(t, expected, newCC.(*collectorCore).loadDelegate().(*bufferedCore).context)
}

func Test_collectorCore_Write(t *testing.T) {
	cc := newCollectorCore(newBufferedCore(zapcore.InfoLevel))
	e := zapcore.Entry{
		Level:   zapcore.DebugLevel,
		Message: "test",
	}
	fields := []zapcore.Field{
		{Key: "field1", String: "value1"},
	}
	err := cc.Write(e, fields)
	require.NoError(t, err)

	expected := loggedEntry{
		e,
		fields,
	}
	require.Len(t, cc.loadDelegate().(*bufferedCore).logs, 1)
	require.Equal(t, expected, cc.loadDelegate().(*bufferedCore).logs[0])
}

func Test_collectorCore_Sync(t *testing.T) {
	cc := newCollectorCore(newBufferedCore(zapcore.InfoLevel))
	assert.NoError(t, cc.Sync())
}

func Test_collectorCore_SetCore(t *testing.T) {
	cc := newCollectorCore(newBufferedCore(zapcore.InfoLevel))
	newCore := newBufferedCore(zapcore.DebugLevel)
	cc.SetCore(newCore)
	assert.Equal(t, zapcore.DebugLevel, cc.loadDelegate().(*bufferedCore).LevelEnabler)
}
