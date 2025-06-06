// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package persistentqueue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch/internal/persistentqueue"

import (
	"context"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/featuregate"
)

func TestBytesMap(t *testing.T) {
	data := []struct {
		key string
		val []byte
	}{
		{"key1", []byte("value1")},
		{"key2", []byte("value2")},
		{"key3", []byte("value3")},
		{"key4", []byte("value4")},
	}

	bm := newBytesMap()
	for _, d := range data {
		err := bm.set(d.key, d.val)
		require.NoError(t, err)
	}

	assert.Equal(t, []string{"key1", "key2", "key3", "key4"}, bm.keys())

	buf, err := bm.get("key2")
	require.NoError(t, err)
	assert.Equal(t, []byte("value2"), buf)
	buf, err = bm.get("key4")
	require.NoError(t, err)
	assert.Equal(t, []byte("value4"), buf)

	buf, err = bm.get("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, buf)

	err = bm.set(strings.Repeat("x", 300), []byte("too long key"))
	require.EqualError(t, err, "key param is too long")

	bm = newBytesMap()

	// invalid key length
	*bm = append(*bm, 4, 'k', 'e', 'y')
	_, err = bm.get("key1")
	require.Error(t, err)
	assert.Empty(t, bm.keys())

	// missing value length
	*bm = append(*bm, '1')
	_, err = bm.get("key1")
	require.Error(t, err)
	assert.Equal(t, []string{"key1"}, bm.keys())

	// missing value
	*bm = binary.LittleEndian.AppendUint32(*bm, 1)
	_, err = bm.get("key1")
	require.Error(t, err)
	assert.Equal(t, []string{"key1"}, bm.keys())
}

func TestBytesMapCarrier(t *testing.T) {
	carrier := &bytesMapCarrier{bytesMap: newBytesMap()}

	carrier.Set("key1", "val1")
	carrier.Set("key2", "val2")

	assert.Equal(t, []string{"key1", "key2"}, carrier.Keys())
	assert.Equal(t, "val2", carrier.Get("key2"))
	assert.Equal(t, "val1", carrier.Get("key1"))
}

func TestEncoder(t *testing.T) {
	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	require.NoError(t, err)
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: 0x01,
		TraceState: trace.TraceState{},
		Remote:     true,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)
	req := uint64(123)
	enc := NewEncoder(Uint64Encoding{})
	fgOrigState := PersistRequestContextFeatureGate.IsEnabled()

	tests := []struct {
		name             string
		fgEnabledOnWrite bool
		fgEnabledOnRead  bool
		wantSpanCtx      trace.SpanContext
		wantReadErr      error
	}{
		{
			name: "feature_gate_disabled_on_write_and_read",
		},
		{
			name:             "feature_gate_enabled_on_write_and_read",
			fgEnabledOnWrite: true,
			fgEnabledOnRead:  true,
			wantSpanCtx:      spanCtx,
		},
		{
			name:            "feature_gate_disabled_on_write_enabled_on_read",
			fgEnabledOnRead: true,
		},
		{
			name:             "feature_gate_enabled_on_write_disabled_on_read",
			fgEnabledOnWrite: true,
			wantReadErr:      ErrInvalidValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(PersistRequestContextFeatureGate.ID(), tt.fgEnabledOnWrite))
			defer func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(PersistRequestContextFeatureGate.ID(), fgOrigState))
			}()
			buf, err := enc.Marshal(ctx, req)
			require.NoError(t, err)

			require.NoError(t, featuregate.GlobalRegistry().Set(PersistRequestContextFeatureGate.ID(), tt.fgEnabledOnRead))
			gotReq, gotCtx, err := enc.Unmarshal(buf)
			assert.Equal(t, tt.wantReadErr, err)
			if err == nil {
				assert.Equal(t, req, gotReq)
				gotSpanCtx := trace.SpanContextFromContext(gotCtx)
				assert.Equal(t, tt.wantSpanCtx, gotSpanCtx)
			}
		})
	}
}
