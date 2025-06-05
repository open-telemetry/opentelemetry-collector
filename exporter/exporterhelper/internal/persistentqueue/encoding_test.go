// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package persistentqueue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/persistentqueue"

import (
	"context"
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

	bm := newBytesMap(0)
	for _, d := range data {
		buf, err := bm.setEmptyBytes(d.key, len(d.val))
		require.NoError(t, err)
		copy(buf, d.val)
	}

	assert.Equal(t, []string{"key1", "key2", "key3", "key4"}, bm.keys())

	buf, err := bm.get("key2")
	require.NoError(t, err)
	assert.Equal(t, []byte("value2"), buf)
	buf, err = bm.get("key4")
	require.NoError(t, err)
	assert.Equal(t, []byte("value4"), buf)
}

func TestBytesMapCarrier(t *testing.T) {
	bm := newBytesMap(0)
	carrier := &bytesMapCarrier{bytesMap: bm}

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
	fgOrigState := persistRequestContextFeatureGate.IsEnabled()

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
			require.NoError(t, featuregate.GlobalRegistry().Set(persistRequestContextFeatureGate.ID(), tt.fgEnabledOnWrite))
			defer func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(persistRequestContextFeatureGate.ID(), fgOrigState))
			}()
			buf, err := enc.Marshal(ctx, req)
			require.NoError(t, err)

			require.NoError(t, featuregate.GlobalRegistry().Set(persistRequestContextFeatureGate.ID(), tt.fgEnabledOnRead))
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
