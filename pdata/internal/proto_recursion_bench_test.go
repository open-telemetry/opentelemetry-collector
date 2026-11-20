// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// benchAnyTree builds a value that alternates ArrayValue and KeyValueList levels. Every
// level has two leaves and two subtrees, so it exercises all four depth-tracked types.
func benchAnyTree(depth int) AnyValue {
	if depth == 0 {
		return AnyValue{Value: &AnyValue_StringValue{StringValue: "leaf-value"}}
	}
	if depth%2 == 0 {
		av := NewArrayValue()
		av.Values = []AnyValue{
			{Value: &AnyValue_IntValue{IntValue: int64(depth)}},
			{Value: &AnyValue_StringValue{StringValue: "str"}},
			benchAnyTree(depth - 1),
			benchAnyTree(depth - 1),
		}
		return AnyValue{Value: &AnyValue_ArrayValue{ArrayValue: av}}
	}
	kvl := NewKeyValueList()
	kvl.Values = []KeyValue{
		{Key: "int", Value: AnyValue{Value: &AnyValue_IntValue{IntValue: int64(depth)}}},
		{Key: "str", Value: AnyValue{Value: &AnyValue_StringValue{StringValue: "str"}}},
		{Key: "left", Value: benchAnyTree(depth - 1)},
		{Key: "right", Value: benchAnyTree(depth - 1)},
	}
	return AnyValue{Value: &AnyValue_KvlistValue{KvlistValue: kvl}}
}

func benchMarshalProto(b *testing.B, size func() int, marshal func([]byte) int) []byte {
	b.Helper()
	buf := make([]byte, size())
	require.Equal(b, len(buf), marshal(buf))
	return buf
}

func BenchmarkUnmarshalProtoAnyValue(b *testing.B) {
	src := benchAnyTree(5)
	buf := benchMarshalProto(b, src.SizeProto, src.MarshalProto)
	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	for b.Loop() {
		require.NoError(b, NewAnyValue().UnmarshalProto(buf))
	}
}

func BenchmarkUnmarshalProtoArrayValue(b *testing.B) {
	src := NewArrayValue()
	for range 8 {
		src.Values = append(src.Values, benchAnyTree(4))
	}
	buf := benchMarshalProto(b, src.SizeProto, src.MarshalProto)
	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	for b.Loop() {
		require.NoError(b, NewArrayValue().UnmarshalProto(buf))
	}
}

func BenchmarkUnmarshalProtoKeyValueList(b *testing.B) {
	src := NewKeyValueList()
	for i := range 8 {
		src.Values = append(src.Values, KeyValue{Key: "k" + strconv.Itoa(i), Value: benchAnyTree(4)})
	}
	buf := benchMarshalProto(b, src.SizeProto, src.MarshalProto)
	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	for b.Loop() {
		require.NoError(b, NewKeyValueList().UnmarshalProto(buf))
	}
}

func BenchmarkUnmarshalProtoKeyValue(b *testing.B) {
	src := &KeyValue{Key: "k", Value: benchAnyTree(5)}
	buf := benchMarshalProto(b, src.SizeProto, src.MarshalProto)
	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	for b.Loop() {
		require.NoError(b, NewKeyValue().UnmarshalProto(buf))
	}
}

// The flat variants measure the per-element cost when there is no nesting at all.
func BenchmarkUnmarshalProtoAnyValueFlat(b *testing.B) {
	src := &AnyValue{Value: &AnyValue_StringValue{StringValue: "v"}}
	buf := benchMarshalProto(b, src.SizeProto, src.MarshalProto)
	b.ReportAllocs()
	for b.Loop() {
		require.NoError(b, NewAnyValue().UnmarshalProto(buf))
	}
}

func BenchmarkUnmarshalProtoKeyValueFlat(b *testing.B) {
	src := &KeyValue{Key: "k", Value: AnyValue{Value: &AnyValue_StringValue{StringValue: "v"}}}
	buf := benchMarshalProto(b, src.SizeProto, src.MarshalProto)
	b.ReportAllocs()
	for b.Loop() {
		require.NoError(b, NewKeyValue().UnmarshalProto(buf))
	}
}
